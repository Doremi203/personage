from abc import ABC, abstractmethod
from app.domain.models.users.UserForGmailProcessingModel import UserForGmailProcessingModel


class IGmailProcessingService(ABC):
    @abstractmethod
    async def get_users_for_processing(self) -> list[UserForGmailProcessingModel]:
        pass

    @abstractmethod
    async def process_users_events(self, users_for_processing: list[UserForGmailProcessingModel]) -> None:
        pass
