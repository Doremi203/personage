from datetime import datetime, timezone

from app.domain.models.events.raw.telegram.RawTelegramMessage import RawTelegramMessage
from app.services.segmentation.album import collapse_albums


def _msg(
    message_id: int,
    text: str = "",
    grouped_id: int | None = None,
    media_kind: str | None = None,
    minutes: int = 0,
    chat_id: int = 1,
    sender_id: int = 100,
) -> RawTelegramMessage:
    return RawTelegramMessage(
        message_id=message_id,
        chat_id=chat_id,
        chat_title="Group",
        chat_type="group",
        sender_id=sender_id,
        sender_username="alice",
        sender_first_name="Alice",
        sender_last_name=None,
        text=text,
        date=datetime(2026, 4, 29, 10, minutes, 0, tzinfo=timezone.utc),
        is_reply=False,
        reply_to_msg_id=None,
        is_forward=False,
        forward_from=None,
        grouped_id=grouped_id,
        media_kind=media_kind,
    )


def test_album_collapses_into_single_buffered_message():
    raws = [
        _msg(101, text="", grouped_id=42, media_kind="photo", minutes=0),
        _msg(102, text="address: ул. Ленина", grouped_id=42, media_kind="photo", minutes=0),
        _msg(103, text="", grouped_id=42, media_kind="photo", minutes=0),
    ]
    out = collapse_albums(raws)
    assert len(out) == 1
    bm = out[0]
    assert sorted(bm.member_ids) == [101, 102, 103]
    assert bm.text == "address: ул. Ленина"
    assert bm.media_kinds == ["photo", "photo", "photo"]
    assert bm.is_noise is False


def test_standalone_messages_pass_through_one_to_one():
    raws = [_msg(1, text="привет"), _msg(2, text="как дела")]
    out = collapse_albums(raws)
    assert len(out) == 2
    assert [b.text for b in out] == ["привет", "как дела"]


def test_album_without_caption_is_kept_as_signal():
    raws = [
        _msg(1, text="", grouped_id=7, media_kind="photo"),
        _msg(2, text="", grouped_id=7, media_kind="photo"),
    ]
    out = collapse_albums(raws)
    assert len(out) == 1
    assert out[0].is_noise is False
    assert out[0].media_kinds == ["photo", "photo"]


def test_sticker_only_message_is_marked_noise():
    raws = [_msg(1, text="", media_kind="sticker")]
    out = collapse_albums(raws)
    assert len(out) == 1
    assert out[0].is_noise is True


def test_messages_returned_in_chronological_order():
    raws = [_msg(2, text="b", minutes=2), _msg(1, text="a", minutes=1)]
    out = collapse_albums(raws)
    assert [b.text for b in out] == ["a", "b"]
