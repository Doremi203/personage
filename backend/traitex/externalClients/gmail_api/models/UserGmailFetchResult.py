from uuid import UUID
from dataclasses import dataclass

from app.domain.models.events.raw.gmail.RawGmailMessage import RawGmailMessage


@dataclass(frozen=True)
class UserGmailFetchResult:
    user_id: UUID | None
    messages: list[RawGmailMessage]
    new_history_id: int | None
    success: bool
    error_message: str | None = None
