import logging
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
        self.logger = logging.getLogger("[CommonProcessingService]")

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
            snapshot_id: SnapshotId,
            target_queue_url: str | None = None
    ) -> int:
        snapshot = await self.snapshot_repository.get_snapshot(snapshot_id)
        if not snapshot:
            raise BusinessException(BusinessErrorCode.ProcessingSnapshotNotFound, f"Processing snapshot with id {snapshot_id} not found")

        max_events_count = 1000
        events = await self.processing_results_repository.get_processing_results(
            from_=snapshot.from_,
            to=snapshot.to,
            limit=max_events_count + 1
        )

        if len(events) > max_events_count:
            self.logger.warning(f"Attempted to resend too many events. Only sent {max_events_count} events, the rest is trimmed")
            events = events[:max_events_count]

        sent_events_count = 0
        try:
            #TODO: batch send to queue?
            for event in events:
                await self.event_producer.send(event, target_queue_url=target_queue_url)
                sent_events_count += 1
        except Exception:
            self.logger.exception(
                "Failed to resend processing snapshot %s after %s/%s events",
                snapshot_id,
                sent_events_count,
                len(events),
            )
            raise

        return len(events)

    async def get_processing_snapshots(self) -> list[SnapshotModel]:
        return await self.snapshot_repository.get_all_snapshots()
