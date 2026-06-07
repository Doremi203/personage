from abc import ABC, abstractmethod
from datetime import datetime
from uuid import UUID


class ITelegramSeenMessageRepository(ABC):
    """Cache of already-processed Telegram messages, keyed by
    (user_id, chat_id, message_id). Used to drop duplicates when the same
    messages are re-fetched on subsequent polling cycles.
    """

    @abstractmethod
    async def get_seen(self, user_id: UUID, pairs: list[tuple[int, int]]) -> set[tuple[int, int]]:
        """Return the subset of (chat_id, message_id) pairs already marked seen."""
        pass

    @abstractmethod
    async def mark_seen(self, user_id: UUID, pairs: list[tuple[int, int]]) -> None:
        """Mark (chat_id, message_id) pairs as seen for the user."""
        pass

    @abstractmethod
    async def delete_seen_before(self, cutoff: datetime) -> None:
        """Drop seen entries older than the cutoff (TTL cleanup)."""
        pass
