from dependency_injector import providers, containers
from dataAccess.infrastructure.PgConnectionProvider import PgConnectionProvider
from messaging.YMQClient import YMQClient


class InfrastructureContainer(containers.DeclarativeContainer):
    config = providers.Configuration()

    pg_connection_provider = providers.Singleton(
        PgConnectionProvider,
        username=config.database.username,
        password=config.database.password,
        host=config.database.host,
        port=config.database.port.as_int(),
        db=config.database.dbname,
        options=config.database.options,
    )

    message_queue = providers.Singleton(
        YMQClient,
        access_key=config.ymq.access_key,
        secret_key=config.ymq.secret_key,
        endpoint_url=config.ymq.endpoint_url,
        default_queue_url=config.ymq.queue_url,
        region=config.ymq.region,
    )
