from abc import ABC, abstractmethod
from uuid import UUID
from dataAccess.models.googleCalendar.UserCalendarProcessingInfo import UserCalendarProcessingInfo


class ICalendarProcessingRepository(ABC):
    @abstractmethod
    async def get_users_processing_info(
            self,
            user_ids: list[UUID]
    ) -> list[UserCalendarProcessingInfo]:
        pass

    @abstractmethod
    async def get_all_users_processing_info(
            self
    ) -> list[UserCalendarProcessingInfo]:
        pass

    @abstractmethod
    async def save_users_processing_info(
            self,
            users_processing_info: list[UserCalendarProcessingInfo]
    ) -> None:
        pass

    @abstractmethod
    async def update_sync_token(self, user_id: UUID, sync_token: str) -> None:
        pass
