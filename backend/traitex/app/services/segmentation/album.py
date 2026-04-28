from app.domain.models.events.raw.telegram.RawTelegramMessage import RawTelegramMessage
from app.services.segmentation.buffer import BufferedMessage
from app.services.segmentation.noise import is_noise_message


def _sender_display(msg: RawTelegramMessage) -> str:
    name = f"{msg.sender_first_name or ''} {msg.sender_last_name or ''}".strip()
    if name:
        return name
    if msg.sender_username:
        return f"@{msg.sender_username}"
    if msg.sender_id is not None:
        return f"user:{msg.sender_id}"
    return "unknown"


def _to_buffered(messages: list[RawTelegramMessage]) -> BufferedMessage:
    """Collapse a list of raw messages (1 plain message OR an album sharing a
    grouped_id) into a single BufferedMessage.
    """
    sorted_messages = sorted(messages, key=lambda m: m.message_id)
    primary: RawTelegramMessage | None = None
    for m in sorted_messages:
        if (m.text or "").strip():
            primary = m
            break
    if primary is None:
        primary = sorted_messages[0]

    text = (primary.text or "").strip()
    media_kinds = [m.media_kind for m in sorted_messages if m.media_kind]

    media_for_noise = media_kinds[0] if len(media_kinds) == 1 and len(sorted_messages) == 1 else (
        "album" if len(sorted_messages) > 1 else None
    )
    noise = is_noise_message(text, media_for_noise)

    return BufferedMessage(
        message_id=primary.message_id,
        member_ids=[m.message_id for m in sorted_messages],
        sender_id=primary.sender_id,
        sender_username=primary.sender_username,
        sender_display=_sender_display(primary),
        text=text,
        date=primary.date,
        grouped_id=primary.grouped_id,
        media_kinds=media_kinds,
        is_noise=noise,
    )


def collapse_albums(messages: list[RawTelegramMessage]) -> list[BufferedMessage]:
    """Group messages by `grouped_id` (within the same chat) and emit one
    BufferedMessage per album / standalone message, ordered by date.

    Telegram sends album members as separate messages with a shared
    `grouped_id`; only one of them carries the caption. Single messages
    (`grouped_id is None`) pass through one-to-one.
    """
    by_group: dict[tuple[int, int], list[RawTelegramMessage]] = {}
    standalone: list[RawTelegramMessage] = []

    for m in messages:
        if m.grouped_id is not None:
            by_group.setdefault((m.chat_id, m.grouped_id), []).append(m)
        else:
            standalone.append(m)

    buffered: list[BufferedMessage] = [_to_buffered(group) for group in by_group.values()]
    buffered.extend(_to_buffered([m]) for m in standalone)
    buffered.sort(key=lambda b: (b.date, b.message_id))
    return buffered
