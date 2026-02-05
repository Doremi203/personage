from dataclasses import dataclass
from datetime import datetime
from uuid import UUID

@dataclass(frozen=True)
class SnapshotModel:
    id: UUID
    from_: datetime
    to: datetime
