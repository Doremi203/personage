from dataclasses import dataclass, field


@dataclass(frozen=True)
class TelegramProcessingCredentialsModel:
    session_string: str
    active_chat_ids: frozenset[int] = field(default_factory=frozenset)
