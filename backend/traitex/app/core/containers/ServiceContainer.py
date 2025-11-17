from dependency_injector import providers, containers


class Services(containers.DeclarativeContainer):
    config = providers.Configuration()
    pass
