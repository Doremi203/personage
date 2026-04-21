from dataclasses import dataclass
from typing import Optional, List
from uuid import UUID

from app.domain.models.events.raw.calendar.RawCalendarEvent import RawCalendarEvent

@dataclass
class UserCalendarFetchResult:
    user_id: UUID
    events: List[RawCalendarEvent]
    new_sync_token: Optional[str]
    success: bool
    error_message: Optional[str] = None
