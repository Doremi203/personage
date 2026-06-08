import hashlib

import grpc
import structlog
from telethon import TelegramClient
from telethon.errors import FloodWaitError
from telethon.sessions import StringSession

from app.config import settings
from app.redis_client import redis_client
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

        cache_key = hashlib.sha256(request.session_string.encode("utf-8")).hexdigest()

        cached = await redis_client.get_chats_cache(cache_key)
        if cached is not None:
            logger.debug("Returning cached Telegram dialogs", count=len(cached))
            return telegram_chats_pb2.GetUserChatsResponse(
                chats=[
                    telegram_chats_pb2.GetUserChatsResponse.Chat(id=int(c["id"]), name=c["name"])
                    for c in cached
                ]
            )

        client = TelegramClient(
            StringSession(request.session_string),
            settings.TELEGRAM_API_ID,
            settings.TELEGRAM_API_HASH,
            flood_sleep_threshold=0,
            device_model=settings.TELEGRAM_DEVICE_MODEL,
            system_version=settings.TELEGRAM_SYSTEM_VERSION,
            app_version=settings.TELEGRAM_APP_VERSION,
            lang_code=settings.TELEGRAM_LANG_CODE,
            system_lang_code=settings.TELEGRAM_SYSTEM_LANG_CODE,
        )

        try:
            await client.connect()

            if not await client.is_user_authorized():
                await context.abort(
                    grpc.StatusCode.UNAUTHENTICATED,
                    "Telegram session is not authorized",
                )

            dialogs = await client.get_dialogs(limit=settings.CHATS_DIALOG_LIMIT)
            chats_payload = [{"id": int(d.id), "name": d.name or ""} for d in dialogs]
            await redis_client.set_chats_cache(
                cache_key, chats_payload, settings.CHATS_CACHE_TTL_SECONDS
            )
            logger.info("Listed Telegram dialogs", count=len(chats_payload))
            return telegram_chats_pb2.GetUserChatsResponse(
                chats=[
                    telegram_chats_pb2.GetUserChatsResponse.Chat(id=c["id"], name=c["name"])
                    for c in chats_payload
                ]
            )
        except FloodWaitError as e:
            logger.warning("Telegram flood wait on GetDialogs", seconds=e.seconds)
            await context.abort(
                grpc.StatusCode.UNAVAILABLE,
                f"Telegram flood wait: retry after {e.seconds}s",
            )
        except grpc.aio.AioRpcError:
            raise
        except Exception as e:
            logger.error("Failed to list Telegram dialogs", error=str(e))
            await context.abort(grpc.StatusCode.UNAVAILABLE, "Failed to list Telegram chats")
        finally:
            await client.disconnect()
