import uuid
from collections import Counter

from app.domain.models.ConnectorTypeModel import ConnectorTypeModel
from app.domain.models.events.enriched.EnrichedEventModel import EnrichedEventModel
from app.domain.models.traits.SenderTrait import SenderTrait
from app.domain.models.traits.base.TraitModel import TraitModel
from app.domain.models.traits.common.UserIdentifier import UserIdentifier
from app.services.segmentation.buffer import BufferedMessage, ConversationSegment


_SEGMENT_NAMESPACE = uuid.UUID("a1d6c4f4-1f15-4a47-9a09-7e1b6f48c0a0")


def _format_media_summary(media_kinds: list[str]) -> str:
    if not media_kinds:
        return ""
    counts = Counter(media_kinds)
    parts = [f"{kind}×{count}" if count > 1 else kind for kind, count in counts.items()]
    return f" [media: {', '.join(parts)}]"


def _format_message_line(seg_first_at, message: BufferedMessage) -> str:
    # Time relative to UTC with seconds precision keeps embeddings stable
    # when the same conversation is replayed from snapshot.
    timestamp = message.date.strftime("%H:%M")
    speaker = message.sender_display or (
        f"@{message.sender_username}" if message.sender_username else "unknown"
    )
    media = _format_media_summary(message.media_kinds)
    text = message.text.strip() if message.text else ""
    if not text and media:
        return f"{speaker} [{timestamp}]{media}"
    if media:
        return f"{speaker} [{timestamp}]{media}: {text}"
    return f"{speaker} [{timestamp}]: {text}"


def render_segment_body(segment: ConversationSegment) -> str:
    if segment.first_at is None:
        return ""
    span_start = segment.first_at.strftime("%Y-%m-%dT%H:%M:%SZ")
    span_end = (segment.last_at or segment.first_at).strftime("%Y-%m-%dT%H:%M:%SZ")
    chat_label = segment.chat_title or f"chat#{segment.chat_id}"
    participant_count = len(segment.participants)
    header = (
        f"[telegram conversation: chat=\"{chat_label}\" type={segment.chat_type} "
        f"messages={segment.message_count} participants={participant_count} "
        f"span={span_start}..{span_end}]"
    )
    sorted_messages = sorted(segment.messages, key=lambda m: (m.date, m.message_id))
    lines = [header]
    for msg in sorted_messages:
        lines.append(_format_message_line(segment.first_at, msg))
    return "\n".join(lines)


def _segment_event_id(segment: ConversationSegment) -> uuid.UUID:
    member_ids = sorted({mid for m in segment.messages for mid in m.member_ids})
    name = (
        f"{segment.user_id}|{segment.chat_id}|"
        f"{member_ids[0] if member_ids else 0}|{member_ids[-1] if member_ids else 0}"
    )
    return uuid.uuid5(_SEGMENT_NAMESPACE, name)


def build_segment_event(segment: ConversationSegment) -> EnrichedEventModel | None:
    """Render a closed segment into an EnrichedEventModel, or None if it has
    no signal-bearing messages (pure noise — should not be emitted).
    """
    primary = segment.first_signal_message
    if primary is None:
        return None

    body = render_segment_body(segment)

    sender_display = primary.sender_display or ""
    traits: list[TraitModel] = [
        SenderTrait(
            identifier=UserIdentifier(
                telegram_id=str(primary.sender_id) if primary.sender_id is not None else None,
                telegram_tag=primary.sender_username,
                telegram_name=sender_display or None,
            )
        )
    ]

    return EnrichedEventModel(
        id=_segment_event_id(segment),
        user_id=segment.user_id,
        connector_type=ConnectorTypeModel.Telegram,
        occurred_at=primary.date,
        main_body=body,
        traits=traits,
    )
