from uuid import UUID
from dataclasses import dataclass


@dataclass
class UserTelegramProcessingInfo:
    user_id: UUID
    last_message_id: int
