from datetime import datetime
from email.policy import default

from pydapper.commands import CommandsAsync

from app.domain.models.processing.SnapshotModel import SnapshotModel
from dataAccess.infrastructure.IPgConnectionProvider import IPgConnectionProvider
from dataAccess.interfaces.IProcessingSnapshotRepository import IProcessingSnapshotRepository, SnapshotId


class ProcessingSnapshotRepository(IProcessingSnapshotRepository):
    def __init__(
            self,
            connection_provider: IPgConnectionProvider
    ):
        self.connection_provider = connection_provider

    async def add_snapshot(self, start: datetime, finish: datetime) -> SnapshotId:
        async with self.connection_provider.get_connection() as commands:
            commands: CommandsAsync

            result = await commands.query_single_async(
                '''
                --ProcessingSnapshotRepository.add_snapshot
                INSERT INTO processing_snapshot(start, finish)
                VALUES (?start?, ?finish?)
                RETURNING id;
                ''',
                param={"start": start, "finish": finish}
            )

            return result['id']

    async def belongs_to_snapshot(self, timestamp: datetime) -> bool:
        async with self.connection_provider.get_connection() as commands:
            commands: CommandsAsync

            return await commands.query_single_or_default_async(
                '''
                --ProcessingSnapshotRepository.belongs_to_snapshot
                SELECT true as belongs_to_snapshot
                FROM processing_snapshot ps
                where ?datetime? between ps.start and ps.finish;
                ''',
                param={"datetime": timestamp},
                model=bool,
                default=False
            )

    async def get_all_snapshots(self) -> list[SnapshotModel]:
        async with self.connection_provider.get_connection() as commands:
            commands: CommandsAsync

            snapshots = await commands.query_async(
                '''
                --ProcessingSnapshotRepository.get_all_snapshots
                SELECT
                    ps.id as id,
                    ps.start as from_,
                    ps.finish as "to"
                FROM processing_snapshot ps;
                ''',
                model=SnapshotModel
            )
            return list(snapshots)

    async def get_snapshot(self, snapshot_id: SnapshotId) -> SnapshotModel | None:
        async with self.connection_provider.get_connection() as commands:
            commands: CommandsAsync

            snapshot = await commands.query_single_or_default_async(
                '''
                --ProcessingSnapshotRepository.get_snapshot
                SELECT
                    ps.id as id,
                    ps.start as from_,
                    ps.finish as "to"
                FROM processing_snapshot ps
                where ps.id = ?id?;
                ''',
                param={"id": snapshot_id},
                model=SnapshotModel,
                default=None
            )
            return snapshot