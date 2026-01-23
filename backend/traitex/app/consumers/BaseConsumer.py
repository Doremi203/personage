from abc import ABC, abstractmethod


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