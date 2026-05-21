import grpc
import structlog
from telethon import TelegramClient
from telethon.sessions import StringSession

from app.config import settings
from proto import telegram_chats_pb2, telegram_chats_pb2_grpc

logger = structlog.get_logger()


class TelegramChatsServicer(telegram_chats_pb2_grpc.TelegramChatsServiceServicer):
    """gRPC service exposing the authenticated user's Telegram dialogs."""

    async def GetUserChats(
        self,
        request: telegram_chats_pb2.GetUserChatsRequest,
        context: grpc.aio.ServicerContext,
    ) -> telegram_chats_pb2.GetUserChatsResponse:
        if not request.session_string:
            await context.abort(grpc.StatusCode.INVALID_ARGUMENT, "session_string is required")

        client = TelegramClient(
            StringSession(request.session_string),
            settings.TELEGRAM_API_ID,
            settings.TELEGRAM_API_HASH,
        )

        try:
            await client.connect()

            if not await client.is_user_authorized():
                await context.abort(
                    grpc.StatusCode.UNAUTHENTICATED,
                    "Telegram session is not authorized",
                )

            dialogs = await client.get_dialogs()
            chats = [
                telegram_chats_pb2.GetUserChatsResponse.Chat(
                    id=int(d.id),
                    name=d.name or "",
                )
                for d in dialogs
            ]
            logger.info("Listed Telegram dialogs", count=len(chats))
            return telegram_chats_pb2.GetUserChatsResponse(chats=chats)
        except grpc.aio.AioRpcError:
            raise
        except Exception as e:
            logger.error("Failed to list Telegram dialogs", error=str(e))
            await context.abort(grpc.StatusCode.UNAVAILABLE, "Failed to list Telegram chats")
        finally:
            await client.disconnect()
