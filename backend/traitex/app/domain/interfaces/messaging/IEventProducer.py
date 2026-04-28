from abc import ABC, abstractmethod
from collections.abc import Iterable
from typing import Any

from app.domain.interfaces.messaging.IQueueClient import BatchResult
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

    @abstractmethod
    async def send_batch(
            self,
            events: Iterable[EnrichedEventModel],
            additional_attributes: dict[str, Any] | None = None,
            target_queue_url: str | None = None
    ) -> BatchResult:
        pass
