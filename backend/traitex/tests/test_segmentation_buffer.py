from datetime import datetime, timezone
from uuid import UUID

from app.services.segmentation.buffer import (
    BufferedMessage,
    SegmentBuffer,
    SegmentationConfig,
)


USER = UUID("11111111-1111-1111-1111-111111111111")
CHAT = 9001


def _bm(
    message_id: int,
    minutes: int,
    text: str = "hi",
    is_noise: bool = False,
    sender_id: int = 1,
) -> BufferedMessage:
    return BufferedMessage(
        message_id=message_id,
        member_ids=[message_id],
        sender_id=sender_id,
        sender_username="alice",
        sender_display="Alice",
        text=text,
        date=datetime(2026, 4, 29, 10, minutes, 0, tzinfo=timezone.utc),
        grouped_id=None,
        media_kinds=[],
        is_noise=is_noise,
    )


def test_silence_window_keeps_segment_until_window_elapses():
    buf = SegmentBuffer(SegmentationConfig(silence_window_seconds=300))
    buf.add(USER, CHAT, "Group", "group", _bm(1, 0))
    buf.add(USER, CHAT, "Group", "group", _bm(2, 2))
    buf.add(USER, CHAT, "Group", "group", _bm(3, 4))

    # 4 minutes elapsed since last message — still inside window
    flushed = buf.flush_stale(datetime(2026, 4, 29, 10, 8, 0, tzinfo=timezone.utc))
    assert flushed == []

    # > 5 minutes after last message — segment closes
    flushed = buf.flush_stale(datetime(2026, 4, 29, 10, 9, 1, tzinfo=timezone.utc))
    assert len(flushed) == 1
    assert flushed[0].message_count == 3


def test_noise_messages_dont_extend_silence_window():
    buf = SegmentBuffer(SegmentationConfig(silence_window_seconds=300))
    buf.add(USER, CHAT, None, "group", _bm(1, 0, text="давай в 19:00"))
    # 4 minutes later: a noise reply ("ок") attaches but does not reset last_at
    buf.add(USER, CHAT, None, "group", _bm(2, 4, text="ок", is_noise=True))

    # 5+ minutes after the original signal-bearing message — should flush
    flushed = buf.flush_stale(datetime(2026, 4, 29, 10, 5, 30, tzinfo=timezone.utc))
    assert len(flushed) == 1
    seg = flushed[0]
    # both messages preserved in transcript
    assert seg.message_count == 2


def test_pure_noise_does_not_open_segment():
    buf = SegmentBuffer(SegmentationConfig())
    seg, overflow = buf.add(USER, CHAT, None, "group", _bm(1, 0, text="ок", is_noise=True))
    assert seg is None
    assert overflow is None
    assert buf.open_segment(USER, CHAT) is None


def test_max_messages_cap_triggers_immediate_flush():
    buf = SegmentBuffer(SegmentationConfig(max_segment_messages=3))
    last_overflow = None
    for i in range(3):
        _, overflow = buf.add(USER, CHAT, None, "group", _bm(i + 1, i))
        last_overflow = overflow
    assert last_overflow is not None
    assert last_overflow.message_count == 3
    # Buffer is empty after overflow flush
    assert buf.open_segment(USER, CHAT) is None


def test_max_span_cap_triggers_immediate_flush():
    buf = SegmentBuffer(SegmentationConfig(max_segment_span_seconds=600))
    buf.add(USER, CHAT, None, "group", _bm(1, 0))
    _, overflow = buf.add(USER, CHAT, None, "group", _bm(2, 11))
    assert overflow is not None
    assert overflow.first_at == datetime(2026, 4, 29, 10, 0, 0, tzinfo=timezone.utc)


def test_separate_chats_buffer_independently():
    buf = SegmentBuffer(SegmentationConfig())
    buf.add(USER, 1, None, "group", _bm(1, 0))
    buf.add(USER, 2, None, "group", _bm(1, 0))

    # Close only chat 1 by waiting past silence window
    seg = buf.force_flush(USER, 1)
    assert seg is not None
    assert buf.open_segment(USER, 2) is not None


def test_flush_all_drains_buffer():
    buf = SegmentBuffer(SegmentationConfig())
    buf.add(USER, 1, None, "group", _bm(1, 0))
    buf.add(USER, 2, None, "group", _bm(1, 0))
    drained = buf.flush_all()
    assert len(drained) == 2
    assert buf.open_segment(USER, 1) is None
    assert buf.open_segment(USER, 2) is None
