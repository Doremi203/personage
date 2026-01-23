from dataclasses import dataclass


@dataclass(frozen=True)
class UserIdentifier:
    telegram_tag: str | None = None
    telegram_name: str | None = None
    email: str | None = None
