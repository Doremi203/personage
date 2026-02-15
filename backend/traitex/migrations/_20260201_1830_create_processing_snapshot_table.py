from pydapper.commands import CommandsAsync
from migrations.infrastructure.MigrationAttribute import *


@migration(version=3, description="Create processing snapshot (id, start, finish)")
class _20260131_1830_create_processing_result_table(ForwardMigration):
    async def migrate_up(self, commands: CommandsAsync) -> None:
        await commands.execute_async(
            sql='''
            CREATE TABLE IF NOT EXISTS processing_snapshot (
                id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                start       TIMESTAMPTZ NOT NULL,
                finish      TIMESTAMPTZ NOT NULL
            );
            '''
        )

        await commands.execute_async(
            sql='''
            CREATE INDEX idx_processing_snapshot_active_period 
            ON processing_snapshot (start, finish);
            '''
        )
