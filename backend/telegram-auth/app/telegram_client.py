# app/telegram_client.py
from telethon import TelegramClient, events
from telethon.sessions import StringSession
from telethon.errors import SessionPasswordNeededError, FloodWaitError
from telethon.tl.types import UpdateShortMessage
import asyncio
import structlog
from typing import Optional, Dict
from app.config import settings

logger = structlog.get_logger()


class TelegramClientManager:
    def __init__(self):
        self.active_clients: Dict[str, TelegramClient] = {}
        self.client_locks: Dict[str, asyncio.Lock] = {}

    async def create_client(self, login_id: str) -> TelegramClient:
        client = TelegramClient(StringSession(), settings.TELEGRAM_API_ID, settings.TELEGRAM_API_HASH)
        await client.connect()

        self.active_clients[login_id] = client
        self.client_locks[login_id] = asyncio.Lock()

        logger.info("Created new Telegram client", login_id=login_id)
        return client

    async def get_client(self, login_id: str) -> Optional[TelegramClient]:
        return self.active_clients.get(login_id)

    async def close_client(self, login_id: str):
        async with self.client_locks.get(login_id, asyncio.Lock()):
            client = self.active_clients.pop(login_id, None)
            if client:
                await client.disconnect()
                logger.info("Closed Telegram client", login_id=login_id)

            self.client_locks.pop(login_id, None)

    async def cleanup_stale_clients(self):
        """Periodic cleanup of stale clients"""
        pass


client_manager = TelegramClientManager()