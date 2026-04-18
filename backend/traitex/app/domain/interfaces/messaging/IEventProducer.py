from abc import ABC, abstractmethod
from typing import Any

from app.domain.models.events.enriched.EnrichedEventModel import EnrichedEventModel


class IEventProducer(ABC):
    @abstractmethod
    async def send(
            self,
            event: EnrichedEventModel,
            additional_attributes: dict[str, Any] | None = None,
            target_queue_url: str | None = None
    ) -> dict[str, Any]:
        pass
