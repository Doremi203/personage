from uuid import UUID
from typing import List

from pydapper.commands import CommandsAsync

from dataAccess.infrastructure.IPgConnectionProvider import IPgConnectionProvider
from dataAccess.interfaces.ITelegramProcessingRepository import ITelegramProcessingRepository
from dataAccess.models.telegram.UserTelegramProcessingInfo import UserTelegramProcessingInfo


class TelegramProcessingRepository(ITelegramProcessingRepository):
    def __init__(self, connection_provider: IPgConnectionProvider):
        self.connection_provider = connection_provider

    async def get_users_processing_info(self, user_ids: List[UUID]) -> List[UserTelegramProcessingInfo]:
        async with self.connection_provider.get_connection() as commands:
            commands: CommandsAsync

            return await commands.query_async(
                '''
                --TelegramProcessingRepository.get_users_processing_info
                SELECT 
                    tp.user_id,
                    tp.last_message_id
                FROM telegram_processing tp
                WHERE tp.user_id = any(?userIds?);
                ''',
                param={"userIds": user_ids},
                model=UserTelegramProcessingInfo
            )

    async def get_all_users_processing_info(self) -> List[UserTelegramProcessingInfo]:
        async with self.connection_provider.get_connection() as commands:
            commands: CommandsAsync

            return await commands.query_async(
                '''
                --TelegramProcessingRepository.get_all_users_processing_info
                SELECT 
                    tp.user_id,
                    tp.last_message_id
                FROM telegram_processing tp;
                ''',
                model=UserTelegramProcessingInfo
            )

    async def save_users_processing_info(self, users_info: List[UserTelegramProcessingInfo]) -> None:
        if not users_info:
            return

        async with self.connection_provider.get_connection() as commands:
            commands: CommandsAsync

            await commands.execute_async(
                '''
                --TelegramProcessingRepository.save_users_processing_info
                INSERT INTO telegram_processing(user_id, last_message_id)
                SELECT
                    unnest(?user_ids?),
                    unnest(?last_message_ids?)
                ON CONFLICT (user_id) 
                DO UPDATE SET
                    last_message_id = excluded.last_message_id;
                ''',
                param={
                    "user_ids": [u.user_id for u in users_info],
                    "last_message_ids": [u.last_message_id for u in users_info]
                }
            )