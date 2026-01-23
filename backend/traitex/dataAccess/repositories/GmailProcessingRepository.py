from uuid import UUID

from pydapper.commands import CommandsAsync

from dataAccess.infrastructure.IPgConnectionProvider import IPgConnectionProvider
from dataAccess.interfaces.IGmailProcessingRepository import IGmailProcessingRepository
from dataAccess.models.gmail.UserProcessingInfo import UserProcessingInfo


class GmailProcessingRepository(IGmailProcessingRepository):
    def __init__(self, connection_provider: IPgConnectionProvider):
        self.connection_provider = connection_provider

    async def get_users_processing_info(self, user_ids: list[UUID]) -> list[UserProcessingInfo]:
        async with self.connection_provider.get_connection() as commands:
            commands: CommandsAsync

            return await commands.query_async(
                '''
                SELECT 
                    gp.user_id,
                    gp.last_message_history_id
                FROM gmail_processing gp
                WHERE gp.user_id = any(?userIds?);
                ''',
                param={"userIds": user_ids},
                model=UserProcessingInfo
            )

    async def save_users_processing_info(self, users_processing_info: list[UserProcessingInfo]) -> None:
        async with self.connection_provider.get_connection() as commands:
            commands: CommandsAsync

            await commands.execute_async(
                '''
                INSERT INTO gmail_processing(user_id, last_message_history_id)
                SELECT
                    unnest(?user_ids?),
                    unnest(?last_history_ids?)
                ON CONFLICT (user_id) 
                DO UPDATE SET
                    last_message_history_id = excluded.last_message_history_id;
                ''',
                param={
                    "user_ids": [u.user_id for u in users_processing_info],
                    "last_history_ids": [u.last_message_history_id for u in users_processing_info]
                }
            )
