from dataclasses import dataclass

from app.domain.models.users.OAuthTokensModel import OAuthTokensModel


@dataclass(frozen=True)
class GmailProcessingCredentialsModel:
    tokens: OAuthTokensModel
