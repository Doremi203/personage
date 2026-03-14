from dataclasses import dataclass

from app.domain.models.users.GmailTokensModel import GmailTokensModel


@dataclass(frozen=True)
class GmailProcessingCredentialsModel:
    tokens: GmailTokensModel
