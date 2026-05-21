import grpc
import structlog

from app.config import settings
from app.telegram_chats_service import TelegramChatsServicer
from proto import telegram_chats_pb2_grpc

logger = structlog.get_logger()


class GrpcServer:
    def __init__(self):
        self._server: grpc.aio.Server | None = None

    async def start(self) -> None:
        if self._server is not None:
            return

        server = grpc.aio.server()
        telegram_chats_pb2_grpc.add_TelegramChatsServiceServicer_to_server(
            TelegramChatsServicer(), server
        )
        listen_addr = f"0.0.0.0:{settings.GRPC_SERVER_PORT}"
        server.add_insecure_port(listen_addr)
        await server.start()
        self._server = server
        logger.info("gRPC server started", address=listen_addr)

    async def stop(self) -> None:
        if self._server is None:
            return
        await self._server.stop(grace=5)
        self._server = None
        logger.info("gRPC server stopped")


grpc_server = GrpcServer()
