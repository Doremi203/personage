from dependency_injector import providers, containers

from app.core.configuration.config import Configuration
from app.core.containers.GrpcServiceContainer import GrpcServiceContainer
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

    grpc_services = providers.Container(
        GrpcServiceContainer,
        services=services,
    )


def create_application_container(config: Configuration) -> ApplicationContainer:
    container = ApplicationContainer()

    container.config.from_dict({
        "database": {
            "username": config.get("Database.Username"),
            "password": config.get("Database.Password"),
            "host": config.get("Database.Host"),
            "port": config.get("Database.Port"),
            "dbname": config.get("Database.Database"),
            "options": config.get("Database.Options", ""),
        },
        "state_tracking": {
            "endpoint": config.get("StateTracking.Endpoint"),
            "use_tls": config.get("StateTracking.UseTls", False)
        },
        "gmail": {
            "max_messages_per_user": config.get("Gmail.MaxMessagesPerUser", 100),
            "client_id": config.get("Gmail.ClientId", ""),
            "client_secret": config.get("Gmail.ClientSecret", "")
        },
        "telegram": {
            "api_id": config.get("Telegram.ApiId"),
            "api_hash": config.get("Telegram.ApiHash"),
        },
        "calendar": {
            "max_events_per_user": config.get("Calendar.MaxEventsPerUser", 100),
            "max_time_window_days": config.get("Calendar.MaxTimeWindowDays", 30)
        },
        "application": {
            "batch_size": config.get("Application.BatchSize", 10),
            "seconds_since_last_process": config.get("Application.SecondsSinceLastProcess", 60)
        },
        "ymq":
            {
                "endpoint_url": config.get("YMQ.EndpointUrl"),
                "queue_url": config.get("YMQ.QueueUrl", config.get("YMQ.EndpointUrl")),
                "access_key": config.get("YMQ.AccessKeyId"),
                "secret_key": config.get("YMQ.SecretAccessKey"),
                "region": config.get("YMQ.Region")
            }
    })

    return container
