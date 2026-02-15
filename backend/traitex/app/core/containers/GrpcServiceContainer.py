import grpc
from dependency_injector import containers, providers
from dependency_injector.containers import DeclarativeContainer
from grpc_reflection.v1alpha import reflection

from app.grpcServices.DebugGrpcService import DebugGrpcService
from proto import (processing_pb2, processing_pb2_grpc,
                   debug_pb2, debug_pb2_grpc)

from app.grpcServices.ProcessingGrpcService import ProcessingGrpcService


class GrpcServiceContainer(containers.DeclarativeContainer):
    services = providers.DependenciesContainer()

    processing_grpc_service = providers.Factory(
        ProcessingGrpcService,
        processing_service=services.common_processing_service,
    )

    debug_grpc_service = providers.Factory(
        DebugGrpcService,
        debug_service=services.debug_service,
    )


def register_grpc_services(container: DeclarativeContainer, server: grpc.Server) -> None:
    processing_grpc_service = container.grpc_services.processing_grpc_service()
    processing_pb2_grpc.add_ProcessingServiceServicer_to_server(processing_grpc_service, server)

    debug_grpc_service = container.grpc_services.debug_grpc_service()
    debug_pb2_grpc.add_DebugServiceServicer_to_server(debug_grpc_service, server)

    add_grpc_reflection(server)


def add_grpc_reflection(server: grpc.aio.server) -> None:
    SERVICE_NAMES = (
        processing_pb2.DESCRIPTOR.services_by_name['ProcessingService'].full_name,
        debug_pb2.DESCRIPTOR.services_by_name['DebugService'].full_name,
        reflection.SERVICE_NAME,
    )
    reflection.enable_server_reflection(SERVICE_NAMES, server)
