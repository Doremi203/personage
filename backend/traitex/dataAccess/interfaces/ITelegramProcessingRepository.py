from abc import ABC, abstractmethod
from uuid import UUID

from dataAccess.models.telegram.UserTelegramProcessingInfo import UserTelegramProcessingInfo


class ITelegramProcessingRepository(ABC):
    @abstractmethod
    async def get_users_processing_info(self, user_ids: list[UUID]) -> list[UserTelegramProcessingInfo]:
        """Get last message IDs for users"""
        pass

    @abstractmethod
    async def save_users_processing_info(self, users_info: list[UserTelegramProcessingInfo]) -> None:
        """Save/update last message IDs for users"""
        pass

    @abstractmethod
    async def get_all_users_processing_info(self) -> list[UserTelegramProcessingInfo]:
        """Get all users with their last message IDs"""
        pass