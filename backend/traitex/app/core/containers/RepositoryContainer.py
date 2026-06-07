from dependency_injector import providers, containers
from dataAccess.repositories.GmailProcessingRepository import GmailProcessingRepository
from dataAccess.repositories.GoogleCalendarProcessingRepository import CalendarProcessingRepository
from dataAccess.repositories.ProcessingResultsRepository import ProcessingResultsRepository
from dataAccess.repositories.ProcessingSnapshotRepository import ProcessingSnapshotRepository
from dataAccess.repositories.TelegramProcessingRepository import TelegramProcessingRepository
from dataAccess.repositories.TelegramSeenMessageRepository import TelegramSeenMessageRepository


class RepositoryContainer(containers.DeclarativeContainer):
    config = providers.Configuration()

    pg_connection_provider = providers.Dependency()

    gmail_processing_repository = providers.Singleton(
        GmailProcessingRepository,
        connection_provider=pg_connection_provider
    )

    telegram_processing_repository = providers.Singleton(
        TelegramProcessingRepository,
        connection_provider=pg_connection_provider
    )

    telegram_seen_message_repository = providers.Singleton(
        TelegramSeenMessageRepository,
        connection_provider=pg_connection_provider
    )

    calendar_processing_repository = providers.Singleton(
        CalendarProcessingRepository,
        connection_provider=pg_connection_provider
    )

    processing_results_repository = providers.Singleton(
        ProcessingResultsRepository,
        connection_provider=pg_connection_provider
    )

    processing_snapshot_repository = providers.Singleton(
        ProcessingSnapshotRepository,
        connection_provider=pg_connection_provider
    )
