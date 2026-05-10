from datetime import datetime
from dataclasses import dataclass


@dataclass
class RawTelegramMessage:
    message_id: int
    chat_id: int
    chat_title: str | None
    chat_type: str
    sender_id: int | None
    sender_username: str | None
    sender_first_name: str | None
    sender_last_name: str | None
    text: str
    date: datetime
    is_reply: bool
    reply_to_msg_id: int | None
    is_forward: bool
    forward_from: str | None
    grouped_id: int | None = None
    media_kind: str | None = None
