from datetime import datetime
from uuid import UUID
from dataclasses import dataclass


@dataclass(frozen=True)
class UserCalendarProcessingInfo:
    user_id: UUID
    last_sync_token: str | None
    last_event_updated_time: datetime | None
