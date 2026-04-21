from abc import ABC, abstractmethod
from app.domain.models.users.UserForProcessingModel import UserForProcessingModel


class ICalendarProcessingService(ABC):
    @abstractmethod
    async def get_users_for_processing(self) -> list[UserForProcessingModel]:
        pass

    @abstractmethod
    async def process_users_events(self, users_for_processing: list[UserForProcessingModel]) -> None:
        pass
