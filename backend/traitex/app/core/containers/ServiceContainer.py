from dependency_injector import providers, containers


class ServiceContainer(containers.DeclarativeContainer):
    config = providers.Configuration()
    pass
