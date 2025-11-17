import asyncio
import logging
from typing import Any

from app.consumers.BaseConsumer import BaseConsumer

logger = logging.getLogger(__name__)

class MockConsumer(BaseConsumer):
    def __init__(self, batch_size: int = 5):
        self.batch_size = batch_size
        self.is_consuming = False
        self.task = None

    @property
    def name(self):
        return "MockConsumer"

    async def start(self) -> None:
        self.is_consuming = True
        self.task = asyncio.create_task(self._consumption_loop())
        logger.info(f"Started {self.name} with batch_size={self.batch_size}")

    async def stop(self) -> None:
        self.is_consuming = False
        if self.task:
            self.task.cancel()
            try:
                await self.task
            except asyncio.CancelledError:
                pass
        logger.info(f"Stopped {self.name}")

    async def process_message(self, message: Any) -> None:
        logger.info(f"Processing message: {message}")
        await asyncio.sleep(0.1)

    async def _consumption_loop(self) -> None:
        while self.is_consuming:
            try:
                mock_message = {"id": 1, "content": "test message"}
                await self.process_message(mock_message)
                await asyncio.sleep(2)

            except asyncio.CancelledError:
                break
            except Exception as e:
                logger.error(f"Consumption loop error: {e}")
                await asyncio.sleep(5)
