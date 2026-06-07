import asyncio
import logging
import traceback

from app.consumers.BaseConsumer import BaseConsumer
from app.domain.interfaces.business_logic.ITelegramProcessingService import ITelegramProcessingService

logger = logging.getLogger(__name__)


class TelegramConsumer(BaseConsumer):
    def __init__(self, telegram_processing_service: ITelegramProcessingService):
        self.telegram_processing_service = telegram_processing_service
        self.is_running = False
        self.task = None

    @property
    def name(self):
        return "TelegramConsumer"

    async def start(self) -> None:
        self.is_running = True
        self.task = asyncio.create_task(self._consumption_loop())
        logger.info(f"Started {self.name}")

    async def stop(self) -> None:
        self.is_running = False
        if self.task:
            self.task.cancel()
            try:
                await self.task
            except asyncio.CancelledError:
                pass
            self.task = None
        logger.info(f"Stopped {self.name}")

    async def _consumption_loop(self) -> None:
        while self.is_running:
            try:
                await self.run_iteration()
                await asyncio.sleep(35)  # Check every 35 seconds
            except asyncio.CancelledError:
                break
            except Exception as e:
                logger.error(f"Error in consumption loop: {e}\n"
                             f"Stack trace: {traceback.format_exc()}")
                await asyncio.sleep(10)

    async def run_iteration(self) -> None:
        users_for_processing = await self.telegram_processing_service.get_users_for_processing()

        if users_for_processing:
            logger.info(f"Start processing Telegram events for {len(users_for_processing)} users")
            await self.telegram_processing_service.process_users_events(users_for_processing)
        else:
            logger.info("No Telegram users found for processing")

        # Drain segments that went idle for users who weren't in this batch.
        # Without this hook, a chat that received its last message right after
        # its owner was processed would wait until the owner re-enters the
        # processing batch (5 min by default) before its segment closes.
        await self.telegram_processing_service.flush_stale_segments()
