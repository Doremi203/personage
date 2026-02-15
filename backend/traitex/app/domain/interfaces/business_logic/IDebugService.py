from uuid import UUID
from abc import ABC, abstractmethod

from app.domain.models.debug.InternalProcessingInfoModel import InternalProcessingInfoModel


class IDebugService(ABC):
    @abstractmethod
    async def rollback_gmail_counter(
            self,
            user_id: UUID,
            decrease_counter_by: int
    ) -> int | None:
        pass

    @abstractmethod
    async def get_processing_info(self) -> list[InternalProcessingInfoModel]:
        pass
