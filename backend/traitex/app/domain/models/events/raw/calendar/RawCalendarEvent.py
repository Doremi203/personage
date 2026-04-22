from dataclasses import dataclass
from datetime import datetime


@dataclass(frozen=True)
class CalendarParticipant:
    email: str
    display_name: str | None = None
    is_organizer: bool = False
    response_status: str | None = None

@dataclass(frozen=True)
class CalendarAttachment:
    filename: str
    file_url: str | None = None
    mime_type: str | None = None

@dataclass(frozen=True)
class RawCalendarEvent:
    id: str
    start_time: datetime
    end_time: datetime
    created_time: datetime
    updated_time: datetime
    summary: str | None = None
    description: str | None = None
    location: str | None = None
    status: str | None = None
    organizer: CalendarParticipant | None = None
    attendees: list[CalendarParticipant] = None
    attachments: list[CalendarAttachment] = None
    recurrence_id: str | None = None
    sequence: int = 0
    hangout_link: str | None = None
