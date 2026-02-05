from datetime import datetime

from app.domain.exceptions.base.BusinessErrorCode import BusinessErrorCode
from app.domain.exceptions.base.BusinessException import BusinessException
from app.domain.interfaces.business_logic.ICommonProcessingService import ICommonProcessingService, SnapshotId
from app.domain.interfaces.messaging.IEventProducer import IEventProducer
from app.domain.models.processing.SnapshotModel import SnapshotModel
from dataAccess.interfaces.IProcessingResultsRepository import IProcessingResultsRepository
from dataAccess.interfaces.IProcessingSnapshotRepository import IProcessingSnapshotRepository


class CommonProcessingService(ICommonProcessingService):
    def __init__(
            self,
            snapshot_repository: IProcessingSnapshotRepository,
            processing_results_repository: IProcessingResultsRepository,
            event_producer: IEventProducer
    ):
        self.snapshot_repository = snapshot_repository
        self.event_producer = event_producer
        self.processing_results_repository = processing_results_repository

    async def create_processing_snapshot(
            self,
            from_: datetime,
            to: datetime
    ) -> SnapshotId:
        snapshot_id = await self.snapshot_repository.add_snapshot(
            start=from_,
            finish=to,
        )
        return snapshot_id

    async def resend_processing_snapshot(
            self,
            snapshot_id: SnapshotId
    ) -> int:
        snapshot = await self.snapshot_repository.get_snapshot(snapshot_id)
        if not snapshot:
            raise BusinessException(BusinessErrorCode.SnapshotNotFound, f"Snapshot with id {snapshot_id} not found")

        max_events_count = 1000
        #TODO: check events count
        events = await self.processing_results_repository.get_processing_results(
            from_=snapshot.from_,
            to=snapshot.to,
        )


    async def get_processing_snapshots(self) -> list[SnapshotModel]:
        return await self.snapshot_repository.get_all_snapshots()
