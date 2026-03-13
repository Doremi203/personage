import grpc
import structlog
from typing import Optional
import asyncio

from app.config import settings
from proto import telegram_pb2, telegram_pb2_grpc

logger = structlog.get_logger()


class AuthServiceGrpcClient:
    """gRPC client for Personage.Auth service"""

    def __init__(self):
        self.channel: Optional[grpc.aio.Channel] = None
        self.stub: Optional[telegram_pb2_grpc.TelegramServiceStub] = None
        self._lock = asyncio.Lock()

    async def connect(self):
        """Establish gRPC connection"""
        async with self._lock:
            if self.channel is not None:
                return

            target = f"{settings.AUTH_SERVICE_GRPC_HOST}:{settings.AUTH_SERVICE_GRPC_PORT}"

            self.channel = grpc.aio.insecure_channel(target)
            logger.info("Created insecure gRPC channel", target=target)


            self.stub = telegram_pb2_grpc.TelegramServiceStub(self.channel)

    async def store_session(self, user_id: str, session_string: str) -> bool:
        """
        Store Telegram session in Personage.Auth via gRPC
        """
        if not self.stub or not self.channel:
            await self.connect()

        try:
            request = telegram_pb2.StoreSessionRequest(
                user_id=user_id,
                session_string=session_string
            )

            response = await self.stub.StoreSession(request)

            if response.success:
                logger.info(
                    "Session stored via gRPC",
                    user_id=user_id,
                    session_id=response.session_id
                )
                return True
            else:
                logger.error(
                    "Failed to store session via gRPC",
                    user_id=user_id,
                    message=response.message
                )
                return False

        except grpc.RpcError as e:
            logger.error(
                "gRPC error storing session",
                user_id=user_id,
                code=e.code().name if hasattr(e, 'code') else 'UNKNOWN',
                details=e.details() if hasattr(e, 'details') else str(e)
            )
            raise
        except Exception as e:
            logger.error("Unexpected error storing session", user_id=user_id, error=str(e))
            raise

    async def get_session(self, user_id: str, include_encrypted: bool = False) -> Optional[str]:
        """
        Retrieve Telegram session from Personage.Auth
        """
        if not self.stub or not self.channel:
            await self.connect()

        try:
            request = telegram_pb2.GetSessionRequest(
                user_id=user_id,
                include_encrypted=include_encrypted
            )

            response = await self.stub.GetSession(request)

            if response.success and response.exists:
                logger.info("Session retrieved via gRPC", user_id=user_id)
                return response.session_string
            else:
                logger.warning("No session found", user_id=user_id)
                return None

        except grpc.RpcError as e:
            logger.error(
                "gRPC error getting session",
                user_id=user_id,
                error=str(e)
            )
            raise

    async def invalidate_session(self, user_id: str, reason: str = None) -> bool:
        """
        Invalidate a user's Telegram session
        """
        if not self.stub or not self.channel:
            await self.connect()

        try:
            request = telegram_pb2.InvalidateSessionRequest(
                user_id=user_id,
                reason=reason or "session_expired"
            )

            response = await self.stub.InvalidateSession(request)

            if response.success:
                logger.info("Session invalidated via gRPC", user_id=user_id)
                return True
            else:
                logger.error("Failed to invalidate session", user_id=user_id)
                return False

        except grpc.RpcError as e:
            logger.error("gRPC error invalidating session", user_id=user_id, error=str(e))
            raise

    async def close(self):
        async with self._lock:
            if self.channel:
                await self.channel.close()
                self.channel = None
                self.stub = None
                logger.info("gRPC channel closed")


auth_service_grpc_client = AuthServiceGrpcClient()
