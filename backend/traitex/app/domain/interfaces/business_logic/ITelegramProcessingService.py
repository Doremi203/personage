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