from uuid import UUID
from dataclasses import dataclass


@dataclass(frozen=True)
class UserGmailProcessingInfo:
    user_id: UUID
    last_message_history_id: int | None
