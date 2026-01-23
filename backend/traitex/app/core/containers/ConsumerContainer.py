from dependency_injector import containers, providers

from app.consumers.GmailConsumer import GmailConsumer


class ConsumerContainer(containers.DeclarativeContainer):
    config = providers.Configuration()

    services = providers.DependenciesContainer()

    gmail_consumer = providers.Singleton(
        GmailConsumer,
        gmail_processing_service=services.gmail_processing_service
    )

    all_consumers = providers.List(
        gmail_consumer,
    )
