from dataclasses import dataclass, field
from datetime import datetime, timedelta
from uuid import UUID


@dataclass
class SegmentationConfig:
    silence_window_seconds: int = 300
    max_segment_messages: int = 50
    max_segment_span_seconds: int = 1800


@dataclass
class BufferedMessage:
    """One logical entry inside a conversation segment.

    Albums (Telegram messages sharing a `grouped_id`) collapse into a single
    BufferedMessage; `member_ids` lists every underlying telegram message_id.
    """
    message_id: int
    member_ids: list[int]
    sender_id: int | None
    sender_username: str | None
    sender_display: str
    text: str
    date: datetime
    grouped_id: int | None
    media_kinds: list[str]
    is_noise: bool


@dataclass
class ConversationSegment:
    user_id: UUID
    chat_id: int
    chat_title: str | None
    chat_type: str
    messages: list[BufferedMessage] = field(default_factory=list)
    first_at: datetime | None = None
    last_at: datetime | None = None

    def add(self, message: BufferedMessage) -> None:
        self.messages.append(message)
        if self.first_at is None or message.date < self.first_at:
            self.first_at = message.date
        if not message.is_noise:
            if self.last_at is None or message.date > self.last_at:
                self.last_at = message.date

    @property
    def message_count(self) -> int:
        return sum(len(m.member_ids) for m in self.messages)

    @property
    def has_signal(self) -> bool:
        return any(not m.is_noise for m in self.messages)

    @property
    def participants(self) -> dict[int, str]:
        out: dict[int, str] = {}
        for m in self.messages:
            if m.sender_id is None:
                continue
            if m.sender_id not in out:
                out[m.sender_id] = m.sender_display
        return out

    @property
    def max_message_id(self) -> int:
        return max(mid for m in self.messages for mid in m.member_ids)

    @property
    def first_signal_message(self) -> BufferedMessage | None:
        for m in self.messages:
            if not m.is_noise:
                return m
        return None


class SegmentBuffer:
    """In-memory buffer of open conversation segments keyed by (user_id, chat_id).

    A segment closes when either:
      - its last non-noise message is older than `silence_window_seconds`, or
      - the cumulative message count crosses `max_segment_messages`, or
      - the span between first and last messages exceeds `max_segment_span_seconds`.
    """

    def __init__(self, config: SegmentationConfig | None = None) -> None:
        self._config = config or SegmentationConfig()
        self._segments: dict[tuple[UUID, int], ConversationSegment] = {}

    @property
    def config(self) -> SegmentationConfig:
        return self._config

    def open_segment(self, user_id: UUID, chat_id: int) -> ConversationSegment | None:
        return self._segments.get((user_id, chat_id))

    def add(
        self,
        user_id: UUID,
        chat_id: int,
        chat_title: str | None,
        chat_type: str,
        message: BufferedMessage,
    ) -> tuple[ConversationSegment | None, ConversationSegment | None]:
        """Add a buffered message to an open segment.

        Returns (segment_after_add, segment_to_emit). `segment_to_emit` is set
        when the addition triggered an immediate flush (size/span overflow).
        Pure-noise messages with no open segment are ignored — caller should
        check the return tuple for `(None, None)`.
        """
        key = (user_id, chat_id)
        seg = self._segments.get(key)
        if seg is None:
            if message.is_noise:
                return None, None
            seg = ConversationSegment(
                user_id=user_id,
                chat_id=chat_id,
                chat_title=chat_title,
                chat_type=chat_type,
            )
            self._segments[key] = seg

        seg.add(message)

        if self._exceeds_caps(seg):
            self._segments.pop(key, None)
            return seg, seg
        return seg, None

    def flush_stale(self, now: datetime) -> list[ConversationSegment]:
        """Pop and return segments whose silence window has elapsed."""
        out: list[ConversationSegment] = []
        for key in list(self._segments.keys()):
            seg = self._segments[key]
            if self._silence_elapsed(seg, now):
                out.append(self._segments.pop(key))
        return out

    def force_flush(self, user_id: UUID, chat_id: int) -> ConversationSegment | None:
        return self._segments.pop((user_id, chat_id), None)

    def flush_all(self) -> list[ConversationSegment]:
        out = list(self._segments.values())
        self._segments.clear()
        return out

    def _silence_elapsed(self, seg: ConversationSegment, now: datetime) -> bool:
        if seg.last_at is None:
            return False
        return (now - seg.last_at) > timedelta(seconds=self._config.silence_window_seconds)

    def _exceeds_caps(self, seg: ConversationSegment) -> bool:
        if seg.message_count >= self._config.max_segment_messages:
            return True
        if seg.first_at is not None and seg.last_at is not None:
            span = seg.last_at - seg.first_at
            if span >= timedelta(seconds=self._config.max_segment_span_seconds):
                return True
        return False
