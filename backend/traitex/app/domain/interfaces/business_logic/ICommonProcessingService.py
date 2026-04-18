import uuid
from abc import ABC, abstractmethod
from datetime import datetime

from app.domain.models.processing.SnapshotModel import SnapshotModel

type SnapshotId = uuid.UUID

class ICommonProcessingService(ABC):
    @abstractmethod
    async def create_processing_snapshot(
            self,
            from_: datetime,
            to: datetime
    ) -> SnapshotId:
        pass

    @abstractmethod
    async def resend_processing_snapshot(
            self,
            snapshot_id: SnapshotId,
            target_queue_url: str | None = None
    ) -> int:
        pass

    @abstractmethod
    async def get_processing_snapshots(self) -> list[SnapshotModel]:
        pass
