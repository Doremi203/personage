from uuid import UUID
from dataclasses import dataclass


@dataclass(frozen=True)
class InternalProcessingInfoModel:
    user_id: UUID
    gmail_counter: int | None
