import asyncio
import base64
import logging
import re
from concurrent.futures import ThreadPoolExecutor
from datetime import datetime
from email.utils import parsedate_to_datetime
from typing import Tuple
from uuid import UUID

import googleapiclient
from google.oauth2.credentials import Credentials
from googleapiclient.discovery import build
from googleapiclient.errors import HttpError

from app.domain.models.events.raw.gmail.EmailAttachment import EmailAttachment
from app.domain.models.events.raw.gmail.EmailParticipant import EmailParticipant
from app.domain.models.events.raw.gmail.RawGmailMessage import RawGmailMessage
from app.domain.models.users.UserForGmailProcessingModel import UserForGmailProcessingModel
from externalClients.gmail_api.models.UserGmailFetchResult import UserGmailFetchResult

logger = logging.getLogger(__name__)


class GmailApiClient:
    def __init__(self, max_messages_per_user: int = 100):
        self.max_messages_per_user = max_messages_per_user
        self._thread_pool = ThreadPoolExecutor(max_workers=10)

    @staticmethod
    def _create_gmail_service(access_token: str) -> googleapiclient.discovery.Resource:
        creds = Credentials(token=access_token)
        return build('gmail', 'v1', credentials=creds, cache_discovery=False)

    @staticmethod
    def _extract_message_body_and_attachments(
            payload: dict
    ) -> Tuple[str, list[EmailAttachment]]:
        body = ""
        attachments = []

        def process_part(part: dict):
            nonlocal body

            mime_type = part.get('mimeType', '').lower()

            if (mime_type == 'text/plain' or mime_type == 'text/html') and not body:
                if 'data' in part.get('body', {}):
                    body = base64.urlsafe_b64decode(part['body']['data']).decode('utf-8')

            elif part.get('filename'):
                attachment = EmailAttachment(
                    filename=part['filename'],
                    mime_type=mime_type,
                    size=int(part.get('body', {}).get('size', 0)),
                    attachment_id=part.get('body', {}).get('attachmentId', '')
                )
                attachments.append(attachment)

            if 'parts' in part:
                for nested_part in part['parts']:
                    process_part(nested_part)

        process_part(payload)

        return body, attachments

    @staticmethod
    def _extract_header_value(headers: list[dict[str, str]], header_name: str) -> str | None:
        for header in headers:
            if header['name'].lower() == header_name.lower():
                return header['value']
        return None

    @staticmethod
    def _parse_email_address(address_str: str) -> list[EmailParticipant]:
        if not address_str:
            return []

        participants = []
        addresses = address_str.split(',')

        for addr in addresses:
            addr = addr.strip()
            if not addr:
                continue

            # "Name <email@domain.com>" format
            match = re.match(r'^(.+?)\s*<(.+?)>$', addr)
            if match:
                name, email = match.groups()
                email = email.strip()

                if email and '@' in email:
                    participants.append(EmailParticipant(
                        name=name.strip() if name else email.split('@')[0],
                        email=email
                    ))
            elif '@' in addr:
                participants.append(EmailParticipant(
                    name=addr.split('@')[0],
                    email=addr.strip()
                ))

        return participants

    @staticmethod
    def _parse_received_date(headers: list[dict[str, str]]) -> datetime:
        date_str = GmailApiClient._extract_header_value(headers, 'date')
        if date_str:
            try:
                return parsedate_to_datetime(date_str)
            except Exception:
                pass

        received_headers = [h['value'] for h in headers if h['name'].lower() == 'received']
        if received_headers:
            last_received = received_headers[-1]
            date_match = re.search(r';\s*([^;]+)$', last_received)
            if date_match:
                try:
                    return parsedate_to_datetime(date_match.group(1).strip())
                except Exception:
                    pass

        return datetime.now()

    @staticmethod
    def _to_raw_gmail_message(message_data: dict) -> RawGmailMessage:
        payload = message_data.get('payload', {})
        headers = payload.get('headers', [])

        subject = GmailApiClient._extract_header_value(headers, 'subject') or '(No Subject)'
        body, attachments = GmailApiClient._extract_message_body_and_attachments(payload)

        received_date = GmailApiClient._parse_received_date(headers)

        from_str = GmailApiClient._extract_header_value(headers, 'from') or ''
        from_participants = GmailApiClient._parse_email_address(from_str)
        from_email = from_participants[0] if from_participants else EmailParticipant(
            name='',
            email='unknown@sender.com'
        )

        to_str = GmailApiClient._extract_header_value(headers, 'to') or ''
        to_emails = GmailApiClient._parse_email_address(to_str)

        labels = message_data.get('labelIds', [])
        history_id = int(message_data.get('historyId', 0))

        return RawGmailMessage(
            id=message_data.get('id', ''),
            body=body,
            subject=subject,
            received_date=received_date,
            from_email=from_email,
            to_emails=to_emails,
            labels=labels,
            history_id=history_id,
            attachments=attachments
        )

    async def fetch_user_messages(
            self,
            user: UserForGmailProcessingModel,
            last_history_id: int | None = None
    ) -> UserGmailFetchResult:
        if not user.tokens:
            return UserGmailFetchResult(
                user_id=user.user_id,
                messages=[],
                new_history_id=last_history_id,
                success=False,
                error_message="No tokens available"
            )

        try:
            service = self._create_gmail_service(user.tokens.access_token)

            if not last_history_id:
                last_history_id = await self._get_user_last_history_id_from_profile(service)

            if not last_history_id:
                return UserGmailFetchResult(
                    user_id=user.user_id,
                    messages=[],
                    new_history_id=last_history_id,
                    success=False,
                    error_message="Unable to fetch last history id"
                )

            return await self._fetch_history_and_messages(service, user, last_history_id)

        except HttpError as e:
            error_msg = f"Gmail API error: {e}"
            logger.error(f"Failed to fetch messages for user {user.user_id}: {error_msg}")

            return UserGmailFetchResult(
                user_id=user.user_id,
                messages=[],
                new_history_id=last_history_id,
                success=False,
                error_message=error_msg
            )
        except Exception as e:
            error_msg = f"Unexpected error: {str(e)}"
            logger.error(f"Unexpected error fetching messages for user {user.user_id}: {e}")

            return UserGmailFetchResult(
                user_id=user.user_id,
                messages=[],
                new_history_id=last_history_id,
                success=False,
                error_message=error_msg
            )

    async def _get_user_last_history_id_from_profile(self, service: googleapiclient.discovery.Resource) -> int | None:
        profile_request = service.users().getProfile(userId='me')
        loop = asyncio.get_event_loop()

        profile_response = await loop.run_in_executor(
            self._thread_pool,
            profile_request.execute
        ) # type: ignore[arg-type]

        if 'historyId' not in profile_response:
            return None
        return int(profile_response['historyId'])

    async def _fetch_history_and_messages(
            self,
            service: googleapiclient.discovery.Resource,
            user: UserForGmailProcessingModel,
            last_history_id: int
    ) -> UserGmailFetchResult:
        """Fetch history records and then full messages"""
        messages = []
        latest_history_id = last_history_id

        try:

            history_response = await self._fetch_history_page(service, last_history_id, None)
            if 'historyId' in history_response:
                latest_history_id = max(latest_history_id, int(history_response['historyId']))

            message_ids = GmailApiClient._extract_message_ids_from_history(history_response)

            if message_ids:
                messages = await self._fetch_full_messages(service, message_ids)

            for history_record in history_response.get('history', []):
                current_id = int(history_record.get('id', 0))
                if current_id > latest_history_id:
                    latest_history_id = current_id

        except Exception as e:
            logger.error(f"Error fetching history for user {user.user_id}: {e}")
            return UserGmailFetchResult(
                user_id=user.user_id,
                messages=messages,
                new_history_id=latest_history_id if messages else last_history_id,
                success=bool(messages),
                error_message=str(e) if not messages else None
            )

        logger.info(
            f"Fetched {len(messages)} messages for user {user.user_id}, "
            f"history: {last_history_id} -> {latest_history_id}"
        )

        return UserGmailFetchResult(
            user_id=user.user_id,
            messages=messages,
            new_history_id=latest_history_id,
            success=True
        )

    async def _fetch_history_page(
            self,
            service: googleapiclient.discovery.Resource,
            start_history_id: int,
            page_token: str | None
    ) -> dict:
        loop = asyncio.get_event_loop()

        params = {
            'userId': 'me',
            'startHistoryId': str(start_history_id),
            'maxResults': self.max_messages_per_user
        }

        if page_token:
            params['pageToken'] = page_token

        history_request = service.users().history().list(**params)
        return await loop.run_in_executor(
            self._thread_pool,
            history_request.execute
        ) # type: ignore[arg-type]

    @staticmethod
    def _extract_message_ids_from_history(history_response: dict) -> list[str]:
        """Extract unique message IDs from history response"""
        message_ids = []

        for history_record in history_response.get('history', []):
            message_fields = ['messages', 'messagesAdded', 'messagesDeleted']

            for field in message_fields:
                if field in history_record:
                    for msg_ref in history_record[field]:
                        if 'message' in msg_ref and 'id' in msg_ref['message']:
                            message_ids.append(msg_ref['message']['id'])
                        elif 'id' in msg_ref:
                            message_ids.append(msg_ref['id'])

        return list(set(message_ids))

    async def _fetch_full_messages(
            self,
            service: googleapiclient.discovery.Resource,
            message_ids: list[str]
    ) -> list[RawGmailMessage]:
        messages = []
        loop = asyncio.get_event_loop()

        for msg_id in message_ids[:self.max_messages_per_user]:
            try:
                msg_request = service.users().messages().get(
                    userId='me',
                    id=msg_id,
                    format='full'
                )

                msg_response = await loop.run_in_executor(
                    self._thread_pool,
                    msg_request.execute
                ) # type: ignore[arg-type]

                raw_message = self._to_raw_gmail_message(msg_response)
                messages.append(raw_message)

            except Exception as e:
                logger.error(f"Failed to fetch message {msg_id}: {e}")
                continue

        return messages

    async def fetch_batch_messages(
            self,
            users_with_last_ids: list[Tuple[UserForGmailProcessingModel, int | None]]
    ) -> dict[UUID, UserGmailFetchResult]:
        tasks = []
        user_map = {}

        for user, last_history_id in users_with_last_ids:
            task = self.fetch_user_messages(user, last_history_id)
            tasks.append(task)
            user_map[task] = user.user_id

        results = await asyncio.gather(*tasks, return_exceptions=True)

        aggregated_results = {}

        for task, result in zip(tasks, results):
            user_id = user_map[task]

            if isinstance(result, Exception):
                logger.error(f"Task failed for user {user_id}: {result}")
                aggregated_results[user_id] = UserGmailFetchResult(
                    user_id=user_id,
                    messages=[],
                    new_history_id=None,
                    success=False,
                    error_message=str(result)
                )
            else:
                aggregated_results[user_id] = result

        return aggregated_results
