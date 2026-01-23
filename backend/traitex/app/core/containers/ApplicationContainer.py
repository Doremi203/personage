from dependency_injector import providers, containers

from app.core.containers.ClientsContainer import ClientsContainer
from app.core.containers.InfrastructureContainer import InfrastructureContainer
from app.core.containers.RepositoryContainer import RepositoryContainer
from app.core.containers.ServiceContainer import ServiceContainer
from app.core.containers.ConsumerContainer import ConsumerContainer


class ApplicationContainer(containers.DeclarativeContainer):
    config = providers.Configuration()

    infrastructure = providers.Container(
        InfrastructureContainer,
        config=config
    )

    repositories = providers.Container(
        RepositoryContainer,
        config=config,
        pg_connection_provider=infrastructure.pg_connection_provider
    )

    clients = providers.Container(
        ClientsContainer,
        config=config
    )

    services = providers.Container(
        ServiceContainer,
        config=config,
        repositories=repositories,
        clients=clients,
        message_queue=infrastructure.message_queue
    )

    consumers = providers.Container(
        ConsumerContainer,
        config=config,
        services=services
    )
