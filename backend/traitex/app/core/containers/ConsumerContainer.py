from dependency_injector import containers, providers

from app.consumers.MockConsumer import MockConsumer


class ConsumerContainer(containers.DeclarativeContainer):
    config = providers.Configuration()

    message_consumer = providers.Singleton(
        MockConsumer,
        batch_size=config.batch_size
    )

    all_consumers = providers.List(
        message_consumer,
    )
