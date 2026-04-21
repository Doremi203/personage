import asyncio
import logging
from concurrent.futures import ThreadPoolExecutor
from datetime import datetime, timezone, timedelta
from typing import Optional, List, Tuple
from uuid import UUID

import googleapiclient
from google.oauth2.credentials import Credentials
from googleapiclient.discovery import build
from googleapiclient.errors import HttpError

from app.domain.models.events.raw.calendar.RawCalendarEvent import RawCalendarEvent, CalendarParticipant, CalendarAttachment
from app.domain.models.users.UserForProcessingModel import UserForProcessingModel
from app.domain.models.users.processingCredentials.CalendarProcessingCredentialsModel import CalendarProcessingCredentialsModel
from externalClients.calendar_api.models.UserCalendarFetchResult import UserCalendarFetchResult

logger = logging.getLogger(__name__)


class CalendarApiClient:
    def __init__(self, max_events_per_user: int = 250, max_time_window_days: int = 30):
        self.max_events_per_user = max_events_per_user
        self.max_time_window_days = max_time_window_days
        self._thread_pool = ThreadPoolExecutor(max_workers=10)

    @staticmethod
    def _create_calendar_service(access_token: str) -> googleapiclient.discovery.Resource:
        creds = Credentials(token=access_token)
        return build('calendar', 'v3', credentials=creds, cache_discovery=False)

    async def fetch_user_events(
            self,
            user: UserForProcessingModel,
            last_sync_token: Optional[str] = None
    ) -> UserCalendarFetchResult:
        if not isinstance(user.credentials, CalendarProcessingCredentialsModel):
            error = "Attempted to fetch calendar info where other processing type was required"
            logger.error(error)
            raise Exception(error)

        calendar_credentials: CalendarProcessingCredentialsModel = user.credentials
        tokens = calendar_credentials.tokens

        if not tokens:
            return UserCalendarFetchResult(
                user_id=user.user_id,
                events=[],
                new_sync_token=last_sync_token,
                success=False,
                error_message="No tokens available"
            )

        service = self._create_calendar_service(tokens.access_token)
        try:
            if last_sync_token:
                return await self._fetch_events_incremental(service, user, last_sync_token)
            else:
                return await self._fetch_events_full_sync(service, user)

        except HttpError as e:
            # Handle 410 GONE - sync token expired
            if e.resp.status == 410:
                logger.warning(f"Sync token expired for user {user.user_id}, falling back to full sync")
                return await self._fetch_events_full_sync(service, user)

            error_msg = f"Calendar API error: {e}"
            logger.error(f"Failed to fetch events for user {user.user_id}: {error_msg}")
            return UserCalendarFetchResult(
                user_id=user.user_id,
                events=[],
                new_sync_token=last_sync_token,
                success=False,
                error_message=error_msg
            )
        except Exception as e:
            error_msg = f"Unexpected error: {str(e)}"
            logger.error(f"Unexpected error fetching events for user {user.user_id}: {e}")
            return UserCalendarFetchResult(
                user_id=user.user_id,
                events=[],
                new_sync_token=last_sync_token,
                success=False,
                error_message=error_msg
            )

    async def _fetch_events_full_sync(
            self,
            service: googleapiclient.discovery.Resource,
            user: UserForProcessingModel
    ) -> UserCalendarFetchResult:
        now = datetime.now(timezone.utc)
        time_min = now.replace(hour=0, minute=0, second=0, microsecond=0)
        time_max = time_min + timedelta(days=self.max_time_window_days)

        time_min_str = time_min.strftime('%Y-%m-%dT%H:%M:%SZ')
        time_max_str = time_max.strftime('%Y-%m-%dT%H:%M:%SZ')

        events = []
        page_token = None
        new_sync_token = None

        try:
            while True:
                loop = asyncio.get_event_loop()
                request = service.events().list(
                    calendarId='primary',
                    timeMin=time_min_str,
                    timeMax=time_max_str,
                    maxResults=self.max_events_per_user,
                    pageToken=page_token,
                    singleEvents=True,
                    orderBy='startTime'
                )

                response = await loop.run_in_executor(
                    self._thread_pool,
                    request.execute
                ) # type: ignore[arg-type]

                if 'nextSyncToken' in response:
                    new_sync_token = response['nextSyncToken']
                elif 'syncToken' in response:
                    new_sync_token = response['syncToken']

                for event in response.get('items', []):
                    raw_event = self._to_raw_calendar_event(event)
                    events.append(raw_event)

                page_token = response.get('nextPageToken')
                if not page_token:
                    break

            logger.info(f"Full sync fetched {len(events)} events for user {user.user_id}")

            return UserCalendarFetchResult(
                user_id=user.user_id,
                events=events,
                new_sync_token=new_sync_token,
                success=True
            )

        except Exception as e:
            logger.error(f"Error during full sync for user {user.user_id}: {e}")
            return UserCalendarFetchResult(
                user_id=user.user_id,
                events=events,
                new_sync_token=None,
                success=False,
                error_message=str(e)
            )

    async def _fetch_events_incremental(
            self,
            service: googleapiclient.discovery.Resource,
            user: UserForProcessingModel,
            sync_token: str
    ) -> UserCalendarFetchResult:
        events = []
        page_token = None
        new_sync_token = sync_token

        try:
            while True:
                loop = asyncio.get_event_loop()
                request = service.events().list(
                    calendarId='primary',
                    syncToken=sync_token,
                    pageToken=page_token,
                    maxResults=self.max_events_per_user
                )

                response = await loop.run_in_executor(
                    self._thread_pool,
                    request.execute
                )

                if 'nextSyncToken' in response:
                    new_sync_token = response['nextSyncToken']

                for event in response.get('items', []):
                    raw_event = self._to_raw_calendar_event(event)
                    events.append(raw_event)

                if 'nextPageToken' not in response:
                    break

                page_token = response.get('nextPageToken')

            logger.info(f"Incremental sync fetched {len(events)} changes for user {user.user_id}")

            deleted_count = sum(1 for e in events if e.status == 'cancelled')
            if deleted_count > 0:
                logger.info(f"  {deleted_count} deleted/cancelled events for user {user.user_id}")

            return UserCalendarFetchResult(
                user_id=user.user_id,
                events=events,
                new_sync_token=new_sync_token,
                success=True
            )

        except Exception as e:
            logger.error(f"Error during incremental sync for user {user.user_id}: {e}")
            return UserCalendarFetchResult(
                user_id=user.user_id,
                events=events,
                new_sync_token=sync_token,
                success=False,
                error_message=str(e)
            )

    @staticmethod
    def _to_raw_calendar_event(event: dict) -> RawCalendarEvent:
        organizer = None
        if event.get('organizer'):
            org = event['organizer']
            organizer = CalendarParticipant(
                email=org.get('email', ''),
                display_name=org.get('displayName'),
                is_organizer=True
            )

        attendees = []
        for attendee in event.get('attendees', []):
            attendees.append(CalendarParticipant(
                email=attendee.get('email', ''),
                display_name=attendee.get('displayName'),
                is_organizer=False,
                response_status=attendee.get('responseStatus')
            ))

        attachments = []
        for attachment in event.get('attachments', []):
            attachments.append(CalendarAttachment(
                filename=attachment.get('title', ''),
                file_url=attachment.get('fileUrl'),
                mime_type=attachment.get('mimeType')
            ))

        start_time = CalendarApiClient._parse_event_time(event.get('start', {}))
        end_time = CalendarApiClient._parse_event_time(event.get('end', {}))

        created_time = datetime.fromisoformat(
            event.get('created', '').replace('Z', '+00:00')
        ) if event.get('created') else datetime.now(timezone.utc)

        updated_time = datetime.fromisoformat(
            event.get('updated', '').replace('Z', '+00:00')
        ) if event.get('updated') else datetime.now(timezone.utc)

        return RawCalendarEvent(
            id=event.get('id', ''),
            summary=event.get('summary'),
            description=event.get('description'),
            location=event.get('location'),
            start_time=start_time,
            end_time=end_time,
            created_time=created_time,
            updated_time=updated_time,
            status=event.get('status', 'confirmed'),
            organizer=organizer,
            attendees=attendees,
            attachments=attachments if attachments else None,
            recurrence_id=event.get('recurringEventId'),
            sequence=event.get('sequence', 0),
            hangout_link=CalendarApiClient._extract_hangout_link(event)
        )

    @staticmethod
    def _extract_hangout_link(event: dict) -> Optional[str]:
        conference_data = event.get('conferenceData', {})
        entry_points = conference_data.get('entryPoints', [])

        for entry in entry_points:
            if entry.get('entryPointType') == 'video':
                return entry.get('uri')
        return None

    @staticmethod
    def _parse_event_time(time_obj: dict) -> datetime:
        if 'dateTime' in time_obj:
            dt_str = time_obj['dateTime']
            return datetime.fromisoformat(dt_str.replace('Z', '+00:00'))
        elif 'date' in time_obj:
            date_str = time_obj['date']
            return datetime.strptime(date_str, '%Y-%m-%d').replace(tzinfo=timezone.utc)
        else:
            return datetime.now(timezone.utc)

    async def fetch_batch_events(
            self,
            users_with_sync_tokens: List[Tuple[UserForProcessingModel, Optional[str]]]
    ) -> dict[UUID, UserCalendarFetchResult]:
        tasks = []
        user_map = {}

        for user, sync_token in users_with_sync_tokens:
            task = self.fetch_user_events(user, sync_token)
            tasks.append(task)
            user_map[task] = user.user_id

        results = await asyncio.gather(*tasks, return_exceptions=True)

        aggregated_results = {}
        for task, result in zip(tasks, results):
            user_id = user_map[task]

            if isinstance(result, Exception):
                logger.error(f"Task failed for user {user_id}: {result}")
                aggregated_results[user_id] = UserCalendarFetchResult(
                    user_id=user_id,
                    events=[],
                    new_sync_token=None,
                    success=False,
                    error_message=str(result)
                )
            else:
                aggregated_results[user_id] = result

        return aggregated_results
