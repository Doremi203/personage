from abc import ABC, abstractmethod
from datetime import datetime
from uuid import UUID

from app.domain.models.processing.SnapshotModel import SnapshotModel

type SnapshotId = UUID


class IProcessingSnapshotRepository(ABC):
    @abstractmethod
    async def add_snapshot(
            self,
            start: datetime,
            finish: datetime
    ) -> SnapshotId:
        pass

    @abstractmethod
    async def belongs_to_snapshot(self, timestamp: datetime) -> bool:
        pass

    @abstractmethod
    async def get_all_snapshots(self) -> list[SnapshotModel]:
        pass

    @abstractmethod
    async def get_snapshot(self, snapshot_id: SnapshotId) -> SnapshotModel | None:
        pass