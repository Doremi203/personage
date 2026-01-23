from dataclasses import dataclass


@dataclass(frozen=True)
class EmailAttachment:
    filename: str
    mime_type: str
    size: int
    attachment_id: str
