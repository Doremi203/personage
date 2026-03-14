from abc import ABC, abstractmethod
from uuid import UUID

from dataAccess.models.gmail.UserGmailProcessingInfo import UserGmailProcessingInfo


class IGmailProcessingRepository(ABC):
    @abstractmethod
    async def get_users_processing_info(
            self,
            user_ids: list[UUID]
    ) -> list[UserGmailProcessingInfo]:
        pass

    @abstractmethod
    async def get_all_users_processing_info(
            self
    ) -> list[UserGmailProcessingInfo]:
        pass

    @abstractmethod
    async def save_users_processing_info(
            self,
            users_processing_info: list[UserGmailProcessingInfo]
    ) -> None:
        pass

    @abstractmethod
    async def decrease_last_history_id(
            self,
            user_id: UUID,
            decrease_by: int
    ) -> int | None:
        pass
