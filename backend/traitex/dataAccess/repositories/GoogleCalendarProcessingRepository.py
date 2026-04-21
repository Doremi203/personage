from datetime import datetime, timezone
from uuid import UUID
from pydapper.commands import CommandsAsync

from dataAccess.infrastructure.IPgConnectionProvider import IPgConnectionProvider
from dataAccess.interfaces.ICalendarProcessingRepository import ICalendarProcessingRepository

from dataAccess.models.googleCalendar.UserCalendarProcessingInfo import UserCalendarProcessingInfo


class CalendarProcessingRepository(ICalendarProcessingRepository):
    def __init__(self, connection_provider: IPgConnectionProvider):
        self.connection_provider = connection_provider

    async def get_users_processing_info(self, user_ids: list[UUID]) -> list[UserCalendarProcessingInfo]:
        async with self.connection_provider.get_connection() as commands:
            commands: CommandsAsync
            return await commands.query_async(
                '''
                --CalendarProcessingRepository.get_users_processing_info
                SELECT 
                    cp.user_id,
                    cp.last_sync_token,
                    cp.last_event_updated_time
                FROM calendar_processing cp
                WHERE cp.user_id = any(?userIds?);
                ''',
                param={"userIds": user_ids},
                model=UserCalendarProcessingInfo
            )

    async def get_all_users_processing_info(self) -> list[UserCalendarProcessingInfo]:
        async with self.connection_provider.get_connection() as commands:
            commands: CommandsAsync
            return await commands.query_async(
                '''
                --CalendarProcessingRepository.get_all_users_processing_info
                SELECT 
                    cp.user_id,
                    cp.last_sync_token,
                    cp.last_event_updated_time
                FROM calendar_processing cp;
                ''',
                model=UserCalendarProcessingInfo
            )

    async def save_users_processing_info(self, users_processing_info: list[UserCalendarProcessingInfo]) -> None:
        async with self.connection_provider.get_connection() as commands:
            commands: CommandsAsync
            await commands.execute_async(
                '''
                --CalendarProcessingRepository.save_users_processing_info
                INSERT INTO calendar_processing(user_id, last_sync_token, last_event_updated_time)
                SELECT
                    unnest(?user_ids?),
                    unnest(?sync_tokens?),
                    unnest(?updated_times?)
                ON CONFLICT (user_id) 
                DO UPDATE SET
                    last_sync_token = excluded.last_sync_token,
                    last_event_updated_time = excluded.last_event_updated_time;
                ''',
                param={
                    "user_ids": [u.user_id for u in users_processing_info],
                    "sync_tokens": [u.last_sync_token for u in users_processing_info],
                    "updated_times": [u.last_event_updated_time for u in users_processing_info]
                }
            )

    async def update_sync_token(self, user_id: UUID, sync_token: str) -> None:
        async with self.connection_provider.get_connection() as commands:
            commands: CommandsAsync
            await commands.execute_async(
                '''
                --CalendarProcessingRepository.update_sync_token
                UPDATE calendar_processing cp
                SET last_sync_token = ?sync_token?,
                    last_event_updated_time = ?updated_time?
                WHERE cp.user_id = ?user_id?;
                ''',
                param={
                    "user_id": user_id,
                    "sync_token": sync_token,
                    "updated_time": datetime.now(timezone.utc)
                }
            )