from uuid import UUID

from pydapper.commands import CommandsAsync

from dataAccess.infrastructure.IPgConnectionProvider import IPgConnectionProvider
from dataAccess.interfaces.IGmailProcessingRepository import IGmailProcessingRepository
from dataAccess.models.gmail.UserGmailProcessingInfo import UserGmailProcessingInfo


class GmailProcessingRepository(IGmailProcessingRepository):
    def __init__(self, connection_provider: IPgConnectionProvider):
        self.connection_provider = connection_provider

    async def get_users_processing_info(self, user_ids: list[UUID]) -> list[UserGmailProcessingInfo]:
        async with self.connection_provider.get_connection() as commands:
            commands: CommandsAsync

            return await commands.query_async(
                '''
                --GmailProcessingRepository.get_users_processing_info
                SELECT 
                    gp.user_id,
                    gp.last_message_history_id
                FROM gmail_processing gp
                WHERE gp.user_id = any(?userIds?);
                ''',
                param={"userIds": user_ids},
                model=UserGmailProcessingInfo
            )

    async def get_all_users_processing_info(
            self
    ) -> list[UserGmailProcessingInfo]:
        async with self.connection_provider.get_connection() as commands:
            commands: CommandsAsync

            return await commands.query_async(
                '''
                --GmailProcessingRepository.get_all_users_processing_info
                SELECT 
                    gp.user_id,
                    gp.last_message_history_id
                FROM gmail_processing gp;
                ''',
                model=UserGmailProcessingInfo
            )

    async def save_users_processing_info(self, users_processing_info: list[UserGmailProcessingInfo]) -> None:
        async with self.connection_provider.get_connection() as commands:
            commands: CommandsAsync

            await commands.execute_async(
                '''
                --GmailProcessingRepository.save_users_processing_info
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

    async def decrease_last_history_id(
            self,
            user_id: UUID,
            decrease_by: int
    ) -> int | None:
        async with self.connection_provider.get_connection() as commands:
            commands: CommandsAsync

            res = await commands.query_single_or_default_async(
                '''
                --GmailProcessingRepository.decrease_last_history_id
                UPDATE gmail_processing gp
                SET last_message_history_id = gp.last_message_history_id - ?decrease_by?
                WHERE gp.user_id = ?user_id?
                RETURNING gp.last_message_history_id;
                ''',
                param={
                    "user_id": user_id,
                    "decrease_by": decrease_by
                },
                default=None
            )

            if res:
                return res['last_message_history_id']
            return None