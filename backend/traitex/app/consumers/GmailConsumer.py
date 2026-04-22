import asyncio
import logging
import traceback

from app.consumers.BaseConsumer import BaseConsumer
from app.domain.interfaces.business_logic.IGmailProcessingService import IGmailProcessingService

logger = logging.getLogger(__name__)


class GmailConsumer(BaseConsumer):
    def __init__(self, gmail_processing_service: IGmailProcessingService):
        self.gmail_processing_service = gmail_processing_service
        self.is_running = False
        self.task = None

    @property
    def name(self):
        return "GmailConsumer"

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
                await asyncio.sleep(30)
            except asyncio.CancelledError:
                break
            except Exception as e:
                logger.error(f"Error in consumption loop: {e}\n"
                             f"Stack trace: {traceback.format_exc()}")
                await asyncio.sleep(10)

    async def run_iteration(self) -> None:
        users_for_processing = await self.gmail_processing_service.get_users_for_processing()

        if len(users_for_processing) == 0:
            logger.info("No users found for processing")
            return

        logger.info(f"Start processing events for {len(users_for_processing)} users")
        await self.gmail_processing_service.process_users_events(users_for_processing)
