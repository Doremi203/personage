from dependency_injector import containers, providers

from app.consumers.GmailConsumer import GmailConsumer
from app.consumers.TelegramConsumer import TelegramConsumer


class ConsumerContainer(containers.DeclarativeContainer):
    config = providers.Configuration()

    services = providers.DependenciesContainer()

    gmail_consumer = providers.Singleton(
        GmailConsumer,
        gmail_processing_service=services.gmail_processing_service
    )

    telegram_consumer = providers.Singleton(
        TelegramConsumer,
        telegram_processing_service=services.telegram_processing_service
    )

    all_consumers = providers.List(
        gmail_consumer,
        telegram_consumer,
    )
