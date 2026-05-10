from abc import ABC, abstractmethod
from typing import List

from app.domain.models.users.UserForProcessingModel import UserForProcessingModel


class ITelegramProcessingService(ABC):
    @abstractmethod
    async def get_users_for_processing(self) -> List[UserForProcessingModel]:
        """Get users who need Telegram message processing"""
        pass

    @abstractmethod
    async def process_users_events(self, users_for_processing: List[UserForProcessingModel]) -> None:
        """Process Telegram messages for the given users"""
        pass

    @abstractmethod
    async def flush_stale_segments(self) -> None:
        """Emit conversation segments whose silence window has elapsed.

        Invoked by the consumer every polling tick so that segments belonging
        to users who are not in the current processing batch still flush in a
        bounded time after the last activity.
        """
        pass