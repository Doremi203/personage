import grpc
from dependency_injector import containers, providers
from proto import processing_pb2_grpc

from app.grpcServices.ProcessingGrpcService import ProcessingGrpcService


class GrpcServiceContainer(containers.DeclarativeContainer):
    services = providers.DependenciesContainer()

    processing_grpc_service = providers.Factory(
        ProcessingGrpcService,
        processing_service=services.processing_service,
    )

    def register_grpc_services(self, server: grpc.Server) -> None:
        processing_grpc_service = self.processing_grpc_service()
        processing_pb2_grpc.add_ProcessingServiceServicer_to_server(processing_grpc_service, server)
