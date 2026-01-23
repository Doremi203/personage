from abc import ABC, abstractmethod
from uuid import UUID

from dataAccess.models.gmail.UserProcessingInfo import UserProcessingInfo


class IGmailProcessingRepository(ABC):
    @abstractmethod
    async def get_users_processing_info(
            self,
            user_ids: list[UUID]
    ) -> list[UserProcessingInfo]:
        pass

    @abstractmethod
    async def save_users_processing_info(
            self,
            users_processing_info: list[UserProcessingInfo]
    ) -> None:
        pass
