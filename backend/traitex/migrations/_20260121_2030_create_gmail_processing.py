from pydapper.commands import CommandsAsync
from migrations.infrastructure.MigrationAttribute import *


@migration(version=1, description="Create gmail processing (id, last_processed_history_id, created_at)")
class _20260121_2030_create_gmail_processing(ForwardMigration):
    async def migrate_up(self, commands: CommandsAsync) -> None:
        await commands.execute_async(
            sql='''
            CREATE TABLE IF NOT EXISTS gmail_processing (
                user_id          UUID PRIMARY KEY,
                last_message_history_id bigint,
                created_at  TIMESTAMPTZ DEFAULT NOW()
            );
            '''
        )
