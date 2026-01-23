from dataclasses import dataclass
from datetime import datetime
from uuid import UUID


@dataclass(frozen=True)
class ProcessedUserModel:
    user_id: UUID
    processed_at: datetime
