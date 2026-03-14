from datetime import datetime
from uuid import UUID
from dataclasses import dataclass
from app.domain.models.users.processingCredentials import ProcessingCredentialsModel


@dataclass(frozen=True)
class UserForProcessingModel:
    user_id: UUID
    last_processed_at: datetime | None
    credentials: ProcessingCredentialsModel
