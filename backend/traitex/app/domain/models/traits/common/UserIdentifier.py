from pydantic import BaseModel


class UserIdentifier(BaseModel):
    telegram_id: str | None = None
    telegram_tag: str | None = None
    telegram_name: str | None = None
    email: str | None = None
    name: str | None = None
