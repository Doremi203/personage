from dependency_injector import providers, containers

from app.core.containers import ServiceContainer, ConsumerContainer


class ApplicationContainer(containers.DeclarativeContainer):
    config = providers.Configuration()

    services = providers.Container(
        ServiceContainer,
        config=config
    )

    consumers = providers.Container(
        ConsumerContainer,
        config=config
    )
