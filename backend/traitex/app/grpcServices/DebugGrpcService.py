from uuid import UUID
from grpc import ServicerContext

from app.domain.interfaces.business_logic.IDebugService import IDebugService
from proto import debug_pb2_grpc, debug_pb2


class DebugGrpcService(debug_pb2_grpc.DebugServiceServicer):
    def __init__(
            self,
            debug_service: IDebugService
    ):
        self.debug_service = debug_service

    async def RollbackGmailProcessing(
            self,
            request: debug_pb2.RollbackGmailProcessingRequest,
            context: ServicerContext
    ) -> debug_pb2.RollbackGmailProcessingResponse:
        res = await self.debug_service.rollback_gmail_counter(
            user_id=UUID(request.user_id),
            decrease_counter_by=request.decrease_processing_counter_by,
        )

        return debug_pb2.RollbackGmailProcessingResponse(
            updated_processing_counter=res
        )

    async def GetInternalProcessingInfo(
            self,
            request: debug_pb2.GetInternalProcessingInfoRequest,
            context: ServicerContext
    ) -> debug_pb2.GetInternalProcessingInfoResponse:
        res = await self.debug_service.get_processing_info()
        return debug_pb2.GetInternalProcessingInfoResponse(
            infos=[debug_pb2.InternalProcessingInfo(
                user_id=str(x.user_id),
                gmail_counter=x.gmail_counter,
            ) for x in res]
        )
