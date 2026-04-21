from dataclasses import dataclass
from datetime import datetime


@dataclass(frozen=True)
class OAuthTokensModel:
    access_token: str
    refresh_token: str
    expires_at: datetime | None
    gmail_email: str
