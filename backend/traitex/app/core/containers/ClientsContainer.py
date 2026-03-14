from dependency_injector import providers, containers
from externalClients.gmail_api.GmailApiClient import GmailApiClient
from externalClients.personage_auth.StateTrackingClient import StateTrackingClient
from externalClients.telegram.TelegramApiClient import TelegramApiClient


class ClientsContainer(containers.DeclarativeContainer):
    config = providers.Configuration()

    state_tracking_client = providers.Singleton(
        StateTrackingClient,
        endpoint=config.state_tracking.endpoint,
        use_tls=config.state_tracking.use_tls.as_(bool)
    )

    gmail_api_client = providers.Singleton(
        GmailApiClient,
        max_messages_per_user=config.gmail.max_messages_per_user.as_int()
    )

    telegram_api_client = providers.Singleton(
        TelegramApiClient,
        api_id=config.telegram.api_id,
        api_hash=config.telegram.api_hash,
    )
