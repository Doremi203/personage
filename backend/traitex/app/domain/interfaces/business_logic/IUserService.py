from abc import ABC, abstractmethod

from app.domain.models.ConnectorTypeModel import ConnectorTypeModel
from app.domain.models.users.ProcessedUserModel import ProcessedUserModel
from app.domain.models.users.UserForGmailProcessingModel import UserForGmailProcessingModel


class IUserService(ABC):
    @abstractmethod
    def get_users_for_processing(self) -> list[UserForGmailProcessingModel]:
        pass

    @abstractmethod
    def mark_users_as_processed(
            self,
            users: list[ProcessedUserModel],
            connector_type: ConnectorTypeModel
    ):
        pass
