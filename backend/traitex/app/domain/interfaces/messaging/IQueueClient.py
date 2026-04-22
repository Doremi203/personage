from abc import ABC, abstractmethod
from typing import Any


class IQueueClient(ABC):
    @abstractmethod
    async def send_single(
            self,
            deduplication_id: str,
            message_group_id: str,
            message_body: str,
            message_attributes: dict[str, Any] | None = None,
            target_queue_url: str | None = None
    ) -> dict[str, Any]:
        pass
