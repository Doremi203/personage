from dependency_injector import providers, containers
from externalClients.gmail_api.GmailApiClient import GmailApiClient
from externalClients.personage_auth.StateTrackingClient import StateTrackingClient


class ClientsContainer(containers.DeclarativeContainer):
    config = providers.Configuration()

    state_tracking_client = providers.Singleton(
        StateTrackingClient,
        endpoint=config.state_tracking.endpoint
    )

    gmail_api_client = providers.Singleton(
        GmailApiClient,
        max_messages_per_user=config.gmail.max_messages_per_user.as_int()
    )
