from datetime import datetime
from uuid import UUID
from dataclasses import dataclass
from app.domain.models.users.GmailTokensModel import GmailTokensModel


@dataclass(frozen=True)
class UserForGmailProcessingModel:
    user_id: UUID
    user_email: str
    tokens: GmailTokensModel | None
    last_processed_at: datetime | None
