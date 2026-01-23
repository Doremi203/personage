from dependency_injector import providers, containers
from dataAccess.repositories.GmailProcessingRepository import GmailProcessingRepository


class RepositoryContainer(containers.DeclarativeContainer):
    config = providers.Configuration()

    pg_connection_provider = providers.Dependency()

    gmail_processing_repository = providers.Singleton(
        GmailProcessingRepository,
        connection_provider=pg_connection_provider
    )
