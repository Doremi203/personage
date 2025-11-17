from abc import ABC, abstractmethod
from typing import Any


class BaseConsumer(ABC):
    """Abstract base class for all message consumers"""

    @abstractmethod
    async def start(self) -> None:
        """Start consuming messages"""
        pass

    @abstractmethod
    async def stop(self) -> None:
        """Stop consuming messages"""
        pass

    @abstractmethod
    async def process_message(self, message: Any) -> None:
        """Process a single message"""
        pass
