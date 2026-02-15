from pydantic import BaseModel


class UserIdentifier(BaseModel):
    telegram_tag: str | None = None
    telegram_name: str | None = None
    email: str | None = None
