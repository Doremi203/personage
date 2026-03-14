from dataclasses import dataclass


@dataclass(frozen=True)
class TelegramProcessingCredentialsModel:
    session_string: str
