from uuid import UUID
from dataclasses import dataclass

from app.domain.models.events.raw.telegram.RawTelegramMessage import RawTelegramMessage


@dataclass
class UserTelegramFetchResult:
    user_id: UUID
    success: bool
    messages: list[RawTelegramMessage]
    new_last_message_id: int | None
    error_message: str | None = None
