from dataclasses import dataclass, field
from datetime import datetime

from app.domain.models.events.raw.gmail.EmailAttachment import EmailAttachment
from app.domain.models.events.raw.gmail.EmailParticipant import EmailParticipant


@dataclass(frozen=True)
class RawGmailMessage:
    id: str
    body: str
    subject: str
    received_date: datetime
    from_email: EmailParticipant
    to_emails: list[EmailParticipant] = field(default_factory=list)
    attachments: list[EmailAttachment] = field(default_factory=list)
    labels: list[str] = field(default_factory=list)
    history_id: int = 0
