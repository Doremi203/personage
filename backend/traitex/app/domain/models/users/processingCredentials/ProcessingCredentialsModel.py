from app.domain.models.users.processingCredentials.GmailProcessingCredentialsModel import \
    GmailProcessingCredentialsModel
from app.domain.models.users.processingCredentials.TelegramProcessingCredentialsModel import \
    TelegramProcessingCredentialsModel

type ProcessingCredentialsModel = GmailProcessingCredentialsModel | TelegramProcessingCredentialsModel