import asyncio
from datetime import datetime, timezone
from uuid import UUID

from app.services.TelegramProcessingService import TelegramProcessingService
from app.services.segmentation import SegmentBuffer, SegmentationConfig
from app.domain.models.events.raw.telegram.RawTelegramMessage import RawTelegramMessage
from externalClients.telegram.models.UserTelegramFetchResult import UserTelegramFetchResult


USER = UUID("11111111-1111-1111-1111-111111111111")
CHAT = 9001


def _msg(message_id: int) -> RawTelegramMessage:
    # Dated "now" so the silence window has not elapsed and the segment stays
    # open in the buffer instead of being emitted — we want to inspect it.
    return RawTelegramMessage(
        message_id=message_id,
        chat_id=CHAT,
        chat_title="Максим",
        chat_type="dm",
        sender_id=1,
        sender_username="maks",
        sender_first_name="Максим",
        sender_last_name=None,
        text=f"message {message_id}",
        date=datetime.now(timezone.utc),
        is_reply=False,
        reply_to_msg_id=None,
        is_forward=False,
        forward_from=None,
    )


def _fetch_result(message_ids: list[int]) -> dict:
    messages = [_msg(mid) for mid in message_ids]
    return {
        USER: UserTelegramFetchResult(
            user_id=USER,
            success=True,
            messages=messages,
            new_last_message_id=max(message_ids),
        )
    }


class _FakeSeenRepo:
    """In-memory stand-in for the postgres seen-message cache."""

    def __init__(self):
        self.seen: set[tuple[UUID, int, int]] = set()

    async def get_seen(self, user_id, pairs):
        return {(c, m) for (c, m) in pairs if (user_id, c, m) in self.seen}

    async def mark_seen(self, user_id, pairs):
        for (c, m) in pairs:
            self.seen.add((user_id, c, m))

    async def delete_seen_before(self, cutoff):
        pass


class _NoDedupSeenRepo(_FakeSeenRepo):
    """Simulates the pre-fix behaviour: nothing is ever remembered as seen."""

    async def get_seen(self, user_id, pairs):
        return set()

    async def mark_seen(self, user_id, pairs):
        pass


class _FakeProcessingRepo:
    async def save_users_processing_info(self, infos):
        pass


class _FakeStateTracking:
    async def mark_processed_users(self, *args, **kwargs):
        pass


class _FakeEventProducer:
    def __init__(self):
        self.sent = []

    async def send(self, event):
        self.sent.append(event)


def _build_service(seen_repo) -> TelegramProcessingService:
    return TelegramProcessingService(
        telegram_processing_repository=_FakeProcessingRepo(),
        telegram_seen_message_repository=seen_repo,
        processing_results_repository=None,
        processing_snapshot_repository=None,
        state_tracking_client=_FakeStateTracking(),
        telegram_api_client=None,
        event_producer=_FakeEventProducer(),
        segment_buffer=SegmentBuffer(SegmentationConfig()),
    )


def _open_segment_message_ids(service) -> list[int]:
    seg = service._segment_buffer.open_segment(USER, CHAT)
    assert seg is not None
    return sorted(mid for m in seg.messages for mid in m.member_ids)


def _run_two_cycles(service):
    active = {USER: frozenset({CHAT})}

    async def scenario():
        # Cycle 1: the conversation starts with two messages.
        await service._process_fetch_results(
            _fetch_result([1, 2]),
            retain_processed_messages=False,
            active_chat_ids_by_user=active,
        )
        # Cycle 2: the global fetch returns the same two messages again plus a
        # new one (the cursor is ignored upstream).
        await service._process_fetch_results(
            _fetch_result([1, 2, 3]),
            retain_processed_messages=False,
            active_chat_ids_by_user=active,
        )

    asyncio.run(scenario())


def test_dedup_keeps_each_message_once_in_open_segment():
    service = _build_service(_FakeSeenRepo())

    _run_two_cycles(service)

    # m1 and m2 from cycle 2 are dropped as already-seen; only m3 is appended.
    assert _open_segment_message_ids(service) == [1, 2, 3]


def test_without_dedup_segment_accumulates_duplicates():
    # Control: with no dedup the buffer re-ingests m1/m2, corrupting the segment.
    service = _build_service(_NoDedupSeenRepo())

    _run_two_cycles(service)

    assert _open_segment_message_ids(service) == [1, 1, 2, 2, 3]
