from datetime import datetime, timezone
from uuid import UUID

from app.services.segmentation.buffer import (
    BufferedMessage,
    ConversationSegment,
)
from app.services.segmentation.renderer import build_segment_event, render_segment_body


USER = UUID("11111111-1111-1111-1111-111111111111")


def _segment_with(messages: list[BufferedMessage], chat_type: str = "group") -> ConversationSegment:
    seg = ConversationSegment(
        user_id=USER,
        chat_id=42,
        chat_title="Family",
        chat_type=chat_type,
    )
    for m in messages:
        seg.add(m)
    return seg


def _bm(
    message_id: int,
    text: str,
    minute: int,
    sender_display: str = "Alice",
    sender_id: int = 1,
    is_noise: bool = False,
    media_kinds: list[str] | None = None,
) -> BufferedMessage:
    return BufferedMessage(
        message_id=message_id,
        member_ids=[message_id],
        sender_id=sender_id,
        sender_username=sender_display.lower(),
        sender_display=sender_display,
        text=text,
        date=datetime(2026, 4, 29, 10, minute, 0, tzinfo=timezone.utc),
        grouped_id=None,
        media_kinds=media_kinds or [],
        is_noise=is_noise,
    )


def test_body_contains_header_and_chronological_messages():
    seg = _segment_with([
        _bm(2, "В 19:30 встречаемся у входа", minute=2),
        _bm(1, "Завтра в 19:00 у Олега?", minute=1),
        _bm(3, "ок", minute=3, is_noise=True),
    ])
    body = render_segment_body(seg)
    lines = body.splitlines()
    assert lines[0].startswith("[telegram conversation: chat=\"Family\" type=group")
    assert "messages=3" in lines[0]
    # span rendered in Moscow local time: 10:01 UTC -> 13:01+03:00
    assert "span=2026-04-29T13:01:00+0300..2026-04-29T13:03:00+0300" in lines[0]
    # 10:0x UTC rendered in Moscow local time (+03:00)
    assert "Alice [13:01]" in lines[1]
    assert "Alice [13:02]" in lines[2]
    assert "Alice [13:03]: ок" in lines[3]


def test_album_renders_with_media_summary():
    bm = BufferedMessage(
        message_id=10,
        member_ids=[10, 11, 12],
        sender_id=2,
        sender_username="bob",
        sender_display="Bob",
        text="address",
        date=datetime(2026, 4, 29, 10, 5, 0, tzinfo=timezone.utc),
        grouped_id=999,
        media_kinds=["photo", "photo", "photo"],
        is_noise=False,
    )
    seg = _segment_with([bm])
    body = render_segment_body(seg)
    assert "[media: photo×3]" in body


def test_build_segment_event_uses_first_signal_sender():
    seg = _segment_with([
        _bm(1, "ок", minute=0, is_noise=True, sender_display="Bob", sender_id=2),
        _bm(2, "встретимся в 19", minute=1, sender_display="Alice", sender_id=1),
    ])
    event = build_segment_event(seg)
    assert event is not None
    assert event.user_id == USER
    assert event.occurred_at == datetime(2026, 4, 29, 10, 1, 0, tzinfo=timezone.utc)
    sender_trait = event.traits[0]
    assert sender_trait.identifier.telegram_id == "1"
    assert sender_trait.identifier.telegram_name == "Alice"


def test_build_segment_event_returns_none_for_pure_noise_segment():
    seg = _segment_with([
        _bm(1, "ок", minute=0, is_noise=True),
    ])
    event = build_segment_event(seg)
    assert event is None


def test_segment_event_id_is_deterministic():
    seg1 = _segment_with([_bm(7, "hi", minute=0), _bm(8, "bye", minute=1)])
    seg2 = _segment_with([_bm(7, "hi", minute=0), _bm(8, "bye", minute=1)])
    e1 = build_segment_event(seg1)
    e2 = build_segment_event(seg2)
    assert e1 is not None and e2 is not None
    assert e1.id == e2.id


def test_segment_event_id_changes_with_message_range():
    seg1 = _segment_with([_bm(7, "hi", minute=0), _bm(8, "bye", minute=1)])
    seg2 = _segment_with([_bm(7, "hi", minute=0), _bm(9, "bye", minute=1)])
    e1 = build_segment_event(seg1)
    e2 = build_segment_event(seg2)
    assert e1 is not None and e2 is not None
    assert e1.id != e2.id
