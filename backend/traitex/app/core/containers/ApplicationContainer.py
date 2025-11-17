from dependency_injector import providers, containers

from app.core.containers.ServiceContainer import ServiceContainer
from app.core.containers.ConsumerContainer import ConsumerContainer


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
