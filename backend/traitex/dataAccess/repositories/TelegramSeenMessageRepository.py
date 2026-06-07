from datetime import datetime
from uuid import UUID

from pydapper.commands import CommandsAsync

from dataAccess.infrastructure.IPgConnectionProvider import IPgConnectionProvider
from dataAccess.interfaces.ITelegramSeenMessageRepository import ITelegramSeenMessageRepository


class TelegramSeenMessageRepository(ITelegramSeenMessageRepository):
    def __init__(self, connection_provider: IPgConnectionProvider):
        self.connection_provider = connection_provider

    async def get_seen(self, user_id: UUID, pairs: list[tuple[int, int]]) -> set[tuple[int, int]]:
        if not pairs:
            return set()

        chat_ids = [p[0] for p in pairs]
        message_ids = [p[1] for p in pairs]

        async with self.connection_provider.get_connection() as commands:
            commands: CommandsAsync

            rows = await commands.query_async(
                '''
                --TelegramSeenMessageRepository.get_seen
                SELECT t.chat_id, t.message_id
                FROM telegram_seen_message t
                JOIN (
                    SELECT unnest(?chat_ids?) AS chat_id,
                           unnest(?message_ids?) AS message_id
                ) q ON q.chat_id = t.chat_id AND q.message_id = t.message_id
                WHERE t.user_id = ?user_id?;
                ''',
                param={
                    "user_id": user_id,
                    "chat_ids": chat_ids,
                    "message_ids": message_ids,
                }
            )

        return {(row["chat_id"], row["message_id"]) for row in rows}

    async def mark_seen(self, user_id: UUID, pairs: list[tuple[int, int]]) -> None:
        if not pairs:
            return

        chat_ids = [p[0] for p in pairs]
        message_ids = [p[1] for p in pairs]

        async with self.connection_provider.get_connection() as commands:
            commands: CommandsAsync

            await commands.execute_async(
                '''
                --TelegramSeenMessageRepository.mark_seen
                INSERT INTO telegram_seen_message(user_id, chat_id, message_id)
                SELECT
                    ?user_id?,
                    unnest(?chat_ids?),
                    unnest(?message_ids?)
                ON CONFLICT (user_id, chat_id, message_id) DO NOTHING;
                ''',
                param={
                    "user_id": user_id,
                    "chat_ids": chat_ids,
                    "message_ids": message_ids,
                }
            )

    async def delete_seen_before(self, cutoff: datetime) -> None:
        async with self.connection_provider.get_connection() as commands:
            commands: CommandsAsync

            await commands.execute_async(
                '''
                --TelegramSeenMessageRepository.delete_seen_before
                DELETE FROM telegram_seen_message
                WHERE seen_at < ?cutoff?;
                ''',
                param={"cutoff": cutoff}
            )
