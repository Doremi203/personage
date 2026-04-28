from abc import ABC, abstractmethod
from dataclasses import dataclass, field
from typing import Any


@dataclass
class QueueEntry:
    id: str
    deduplication_id: str
    message_group_id: str
    message_body: str
    message_attributes: dict[str, Any] | None = None


@dataclass
class BatchSendFailure:
    id: str
    code: str
    message: str
    sender_fault: bool


@dataclass
class BatchResult:
    successful_ids: list[str] = field(default_factory=list)
    failed: list[BatchSendFailure] = field(default_factory=list)


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

    @abstractmethod
    async def send_batch(
            self,
            entries: list[QueueEntry],
            target_queue_url: str | None = None
    ) -> BatchResult:
        pass
