from pydapper.commands import CommandsAsync
from migrations.infrastructure.MigrationAttribute import *


@migration(version=2, description="Create processing result (id, processed_at, is_retained, event)")
class _20260131_1830_create_processing_result_table(ForwardMigration):
    async def migrate_up(self, commands: CommandsAsync) -> None:
        await commands.execute_async(
            sql='''
            CREATE TABLE IF NOT EXISTS processing_result (
                id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                processed_at    TIMESTAMPTZ NOT NULL,
                event           JSONB NOT NULL
            );
            '''
        )

        await commands.execute_async(
            sql='''
            CREATE INDEX idx_processing_result_processed_at
            ON processing_result (processed_at) 
            '''
        )
