import logging
import asyncio
from uuid import UUID
from datetime import datetime, timezone

from telethon import TelegramClient
from telethon.errors import FloodWaitError
from telethon.sessions import StringSession
from telethon.tl.types import Message, User, Chat, Channel

from app.domain.models.events.raw.telegram.RawTelegramMessage import RawTelegramMessage
from app.domain.models.users.UserForProcessingModel import UserForProcessingModel
from externalClients.telegram.models.UserTelegramFetchResult import UserTelegramFetchResult


class TelegramApiClient:
    def __init__(
            self,
            api_id: str,
            api_hash: str
    ):
        self.api_id = int(api_id)
        self.api_hash = api_hash
        self.logger = logging.getLogger("[TelegramApiClient]")
        self._client_cache = {}

    async def fetch_batch_messages(
            self,
            users_with_last_ids: list[tuple[UserForProcessingModel, int | None]]
    ) -> dict[UUID, UserTelegramFetchResult]:
        """Fetch messages for multiple users in parallel with concurrency control"""
        semaphore = asyncio.Semaphore(3)

        async def fetch_single(user: UserForProcessingModel, last_id: int | None):
            async with semaphore:
                return await self._fetch_user_messages(user, last_id)

        tasks = [
            fetch_single(user, last_id)
            for user, last_id in users_with_last_ids
        ]

        results_list = await asyncio.gather(*tasks, return_exceptions=True)

        results_dict = {}
        for i, result in enumerate(results_list):
            user_id = users_with_last_ids[i][0].user_id

            if isinstance(result, Exception):
                self.logger.error(f"Failed to fetch for user {user_id}: {result}")
                results_dict[user_id] = UserTelegramFetchResult(
                    user_id=user_id,
                    success=False,
                    messages=[],
                    new_last_message_id=None,
                    error_message=str(result)
                )
            else:
                results_dict[user_id] = result

        return results_dict

    async def _fetch_user_messages(
            self,
            user: UserForProcessingModel,
            last_id: int | None
    ) -> UserTelegramFetchResult:
        """Fetch messages for a single user"""

        try:
            client = await self._get_client(user)

            limit = 100
            if last_id is None:
                messages = await self._fetch_messages_safely(client, limit=limit)
                new_last_id = messages[0].id if messages else None
            else:
                messages = await self._fetch_messages_safely(
                    client,
                    min_id=last_id
                )
                messages.reverse()

                new_last_id = max([m.id for m in messages]) if messages else last_id

            raw_messages = [
                TelegramApiClient.__convert_to_raw_message(msg)
                for msg in messages
            ]

            return UserTelegramFetchResult(
                user_id=user.user_id,
                success=True,
                messages=raw_messages,
                new_last_message_id=new_last_id
            )

        except Exception as e:
            self.logger.error(f"Error fetching for user {user.user_id}: {e}")
            return UserTelegramFetchResult(
                user_id=user.user_id,
                success=False,
                messages=[],
                new_last_message_id=None,
                error_message=str(e)
            )
        finally:
            pass

    async def _fetch_messages_safely(self, client: TelegramClient, **kwargs):
        """Fetch messages with retry logic for flood waits"""
        max_retries = 3
        retry_delay = 5

        for attempt in range(max_retries):
            try:
                messages = []
                async for msg in client.iter_messages(
                        None,
                        wait_time=2,
                        **kwargs
                ):
                    if msg.text:
                        messages.append(msg)

                    if len(messages) >= 100:
                        break

                return messages

            except FloodWaitError as e:
                wait_time = e.seconds
                self.logger.warning(f"Flood wait for {wait_time} seconds")
                if attempt < max_retries - 1:
                    await asyncio.sleep(min(wait_time, 30))
                else:
                    raise
            except Exception:
                if attempt < max_retries - 1:
                    await asyncio.sleep(retry_delay)
                else:
                    raise

        return []

    async def _get_client(self, user: UserForProcessingModel) -> TelegramClient:
        """Get or create a Telegram client for a user"""

        if user.user_id in self._client_cache:
            client = self._client_cache[user.user_id]
            if client.is_connected():
                return client
            else:
                await client.connect()
                return client

        # Create new client
        client = TelegramClient(
            StringSession(user.credentials.session_string),
            self.api_id,
            self.api_hash
        )

        await client.connect()

        # Verify the session works
        if not await client.is_user_authorized():
            raise Exception(f"Session invalid for user {user.user_id}")

        self._client_cache[user.user_id] = client
        return client

    @staticmethod
    def __convert_to_raw_message(msg: Message) -> RawTelegramMessage:
        """Convert Telethon Message to our domain model"""
        chat_title = None
        chat_id = None
        if msg.chat:
            chat_id = msg.chat.id
            if isinstance(msg.chat, (Chat, Channel)):
                chat_title = msg.chat.title
            elif isinstance(msg.chat, User):
                chat_title = f"{msg.chat.first_name or ''} {msg.chat.last_name or ''}".strip()

        sender_id = None
        sender_username = None
        sender_first_name = None
        sender_last_name = None

        if msg.sender:
            sender_id = msg.sender.id
            if isinstance(msg.sender, User):
                sender_username = msg.sender.username
                sender_first_name = msg.sender.first_name
                sender_last_name = msg.sender.last_name

        forward_from = None
        if msg.forward:
            if msg.forward.sender:
                forward_from = f"@{msg.forward.sender.username}" if msg.forward.sender.username else "Unknown"

        return RawTelegramMessage(
            message_id=msg.id,
            chat_id=chat_id or 0,
            chat_title=chat_title,
            sender_id=sender_id,
            sender_username=sender_username,
            sender_first_name=sender_first_name,
            sender_last_name=sender_last_name,
            text=msg.text or "",
            date=msg.date.replace(tzinfo=timezone.utc) if msg.date else datetime.now(timezone.utc),
            is_reply=msg.reply_to is not None,
            reply_to_msg_id=msg.reply_to.reply_to_msg_id if msg.reply_to else None,
            is_forward=msg.forward is not None,
            forward_from=forward_from
        )
