from abc import ABC, abstractmethod
from datetime import datetime

from app.domain.models.events.enriched.EnrichedEventModel import EnrichedEventModel


class IProcessingResultsRepository(ABC):
    @abstractmethod
    async def save_processing_result(
            self,
            processed_at: datetime,
            processing_result: EnrichedEventModel
    ) -> None:
        pass

    @abstractmethod
    async def get_processing_results(
            self,
            from_: datetime,
            to: datetime,
            limit: int
    ) -> list[EnrichedEventModel]:
        pass
