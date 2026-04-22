from dependency_injector import providers, containers

from app.services.CommonProcessingService import CommonProcessingService
from app.services.DebugService import DebugService
from app.services.GmailProcessingService import GmailProcessingService
from app.services.GoogleCalendarProcessingService import CalendarProcessingService
from app.services.TelegramProcessingService import TelegramProcessingService
from messaging.EventProducer import EventProducer


class ServiceContainer(containers.DeclarativeContainer):
    config = providers.Configuration()
    repositories = providers.DependenciesContainer()
    clients = providers.DependenciesContainer()
    message_queue = providers.Dependency()

    event_producer = providers.Factory(
        EventProducer,
        queue_client=message_queue
    )

    gmail_processing_service = providers.Factory(
        GmailProcessingService,
        gmail_processing_repository=repositories.gmail_processing_repository,
        state_tracking_client=clients.state_tracking_client,
        gmail_api_client=clients.gmail_api_client,
        event_producer=event_producer,
        processing_results_repository=repositories.processing_results_repository,
        processing_snapshot_repository=repositories.processing_snapshot_repository,
    )

    telegram_processing_service = providers.Factory(
        TelegramProcessingService,
        telegram_processing_repository=repositories.telegram_processing_repository,
        processing_results_repository=repositories.processing_results_repository,
        processing_snapshot_repository=repositories.processing_snapshot_repository,
        state_tracking_client=clients.state_tracking_client,
        telegram_api_client=clients.telegram_api_client,
        event_producer=event_producer
    )

    calendar_processing_service = providers.Factory(
        CalendarProcessingService,
        calendar_processing_repository=repositories.calendar_processing_repository,
        processing_results_repository=repositories.processing_results_repository,
        processing_snapshot_repository=repositories.processing_snapshot_repository,
        state_tracking_client=clients.state_tracking_client,
        calendar_api_client=clients.calendar_api_client,
        event_producer=event_producer,
    )

    common_processing_service = providers.Factory(
        CommonProcessingService,
        snapshot_repository=repositories.processing_snapshot_repository,
        processing_results_repository=repositories.processing_results_repository,
        event_producer=event_producer,
    )

    debug_service = providers.Factory(
        DebugService,
        gmail_processing_repository=repositories.gmail_processing_repository,
    )
