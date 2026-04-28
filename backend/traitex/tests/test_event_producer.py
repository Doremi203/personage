import base64
import datetime
import uuid

import pytest

from app.domain.exceptions.base.BusinessErrorCode import BusinessErrorCode
from app.domain.exceptions.base.BusinessException import BusinessException
from app.domain.interfaces.messaging.IQueueClient import (
    BatchResult,
    BatchSendFailure,
    IQueueClient,
    QueueEntry,
)
from app.domain.models.ConnectorTypeModel import ConnectorTypeModel
from app.domain.models.events.enriched.EnrichedEventModel import EnrichedEventModel
from app.domain.models.traits.SubjectTrait import SubjectTrait
from messaging.EventProducer import EventProducer


class FakeQueueClient(IQueueClient):
    def __init__(self) -> None:
        self.singles: list[dict] = []
        self.batches: list[list[QueueEntry]] = []
        self.batch_response: BatchResult | None = None

    async def send_single(
            self,
            deduplication_id,
            message_group_id,
            message_body,
            message_attributes=None,
            target_queue_url=None,
    ):
        self.singles.append({
            'deduplication_id': deduplication_id,
            'message_group_id': message_group_id,
            'message_body': message_body,
            'message_attributes': message_attributes,
            'target_queue_url': target_queue_url,
        })
        return {'MessageId': 'fake'}

    async def send_batch(self, entries, target_queue_url=None):
        self.batches.append(list(entries))
        if self.batch_response is not None:
            return self.batch_response
        return BatchResult(successful_ids=[e.id for e in entries])


def _make_event(
        body: str = "hello",
        subject: str = "Hi",
        user_id: uuid.UUID | None = None,
        event_id: uuid.UUID | None = None,
        occurred_at: datetime.datetime | None = None,
) -> EnrichedEventModel:
    return EnrichedEventModel(
        id=event_id or uuid.uuid4(),
        user_id=user_id or uuid.UUID("11111111-1111-1111-1111-111111111111"),
        connector_type=ConnectorTypeModel.Gmail,
        occurred_at=occurred_at or datetime.datetime(2025, 1, 1, 12, 0, 0, tzinfo=datetime.UTC),
        main_body=body,
        traits=[SubjectTrait(name=subject)],
    )


@pytest.mark.asyncio
async def test_dedup_id_is_stable_across_random_event_ids():
    user_id = uuid.UUID("22222222-2222-2222-2222-222222222222")
    occurred = datetime.datetime(2025, 4, 29, 10, 0, 0, tzinfo=datetime.UTC)

    e1 = _make_event(body="same", subject="same", user_id=user_id, occurred_at=occurred, event_id=uuid.uuid4())
    e2 = _make_event(body="same", subject="same", user_id=user_id, occurred_at=occurred, event_id=uuid.uuid4())

    client = FakeQueueClient()
    producer = EventProducer(client)
    await producer.send(e1)
    await producer.send(e2)

    assert e1.id != e2.id
    assert client.singles[0]['deduplication_id'] == client.singles[1]['deduplication_id']


@pytest.mark.asyncio
async def test_dedup_id_changes_with_content():
    e1 = _make_event(body="version 1")
    e2 = _make_event(body="version 2")

    client = FakeQueueClient()
    producer = EventProducer(client)
    await producer.send(e1)
    await producer.send(e2)

    assert client.singles[0]['deduplication_id'] != client.singles[1]['deduplication_id']


@pytest.mark.asyncio
async def test_dedup_id_changes_with_connector():
    occurred = datetime.datetime(2025, 4, 29, 10, 0, 0, tzinfo=datetime.UTC)
    user_id = uuid.UUID("44444444-4444-4444-4444-444444444444")

    e_gmail = _make_event(body="x", subject="x", user_id=user_id, occurred_at=occurred)
    e_telegram = _make_event(body="x", subject="x", user_id=user_id, occurred_at=occurred)
    e_telegram.connector_type = ConnectorTypeModel.Telegram
    e_calendar = _make_event(body="x", subject="x", user_id=user_id, occurred_at=occurred)
    e_calendar.connector_type = ConnectorTypeModel.GoogleCalendar

    client = FakeQueueClient()
    producer = EventProducer(client)
    await producer.send(e_gmail)
    await producer.send(e_telegram)
    await producer.send(e_calendar)

    dedup_ids = {s['deduplication_id'] for s in client.singles}
    assert len(dedup_ids) == 3, "dedup ids must differ across connectors"


@pytest.mark.asyncio
async def test_dedup_id_changes_with_user():
    occurred = datetime.datetime(2025, 4, 29, 10, 0, 0, tzinfo=datetime.UTC)
    e1 = _make_event(body="x", subject="x", user_id=uuid.UUID("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"), occurred_at=occurred)
    e2 = _make_event(body="x", subject="x", user_id=uuid.UUID("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"), occurred_at=occurred)

    client = FakeQueueClient()
    producer = EventProducer(client)
    await producer.send(e1)
    await producer.send(e2)

    assert client.singles[0]['deduplication_id'] != client.singles[1]['deduplication_id']


@pytest.mark.asyncio
async def test_oversized_body_is_truncated_to_fit_ymq_limit():
    huge_body = "Ы" * (300 * 1024)  # ~600 KiB UTF-8
    event = _make_event(body=huge_body)

    client = FakeQueueClient()
    producer = EventProducer(client)
    await producer.send(event)

    sent = client.singles[0]
    body_b64 = sent['message_body']
    assert len(body_b64.encode('utf-8')) <= 256 * 1024
    raw_proto_bytes = base64.b64decode(body_b64)
    assert len(raw_proto_bytes) <= EventProducer._MAX_PROTO_BYTES

    from proto import events_pb2
    decoded = events_pb2.Event()
    decoded.ParseFromString(raw_proto_bytes)
    assert decoded.context.body.endswith(EventProducer._TRUNCATION_MARKER)


@pytest.mark.asyncio
async def test_truncated_events_with_same_truncation_share_dedup_id():
    huge1 = ("a" * (200 * 1024)) + "TAIL ONE"
    huge2 = ("a" * (200 * 1024)) + "TAIL TWO"

    occurred = datetime.datetime(2025, 4, 29, 10, 0, 0, tzinfo=datetime.UTC)
    user_id = uuid.UUID("33333333-3333-3333-3333-333333333333")

    e1 = _make_event(body=huge1, user_id=user_id, occurred_at=occurred)
    e2 = _make_event(body=huge2, user_id=user_id, occurred_at=occurred)

    client = FakeQueueClient()
    producer = EventProducer(client)
    await producer.send(e1)
    await producer.send(e2)

    # Both bodies are identical for the first ~189 KiB and only differ in the
    # part that gets truncated, so the post-truncation payloads (and therefore
    # their dedup ids) must match.
    assert client.singles[0]['deduplication_id'] == client.singles[1]['deduplication_id']


@pytest.mark.asyncio
async def test_send_batch_uses_queue_send_batch():
    events = [_make_event(body=f"body-{i}") for i in range(3)]
    client = FakeQueueClient()
    producer = EventProducer(client)

    result = await producer.send_batch(events)

    assert len(client.batches) == 1
    assert len(client.batches[0]) == 3
    assert len(result.successful_ids) == 3
    # Dedup ids must all be different (different content).
    dedup_ids = {entry.deduplication_id for entry in client.batches[0]}
    assert len(dedup_ids) == 3


@pytest.mark.asyncio
async def test_send_batch_passes_failures_through():
    events = [_make_event(body=f"body-{i}") for i in range(3)]
    client = FakeQueueClient()
    client.batch_response = BatchResult(
        successful_ids=[str(events[0].id)[:80], str(events[2].id)[:80]],
        failed=[BatchSendFailure(
            id=str(events[1].id)[:80],
            code='InternalError',
            message='boom',
            sender_fault=False,
        )],
    )
    producer = EventProducer(client)

    result = await producer.send_batch(events)

    assert len(result.successful_ids) == 2
    assert len(result.failed) == 1
    assert result.failed[0].code == 'InternalError'


@pytest.mark.asyncio
async def test_send_batch_drops_unshippable_event_before_calling_client():
    # Simulate an event that would still be too big to fit even after truncation
    # by patching the size limit down to something tiny.
    original_limit = EventProducer._MAX_PROTO_BYTES
    EventProducer._MAX_PROTO_BYTES = 1  # too small for anything to fit
    try:
        events = [_make_event(body="anything")]
        client = FakeQueueClient()
        producer = EventProducer(client)
        # send should still raise BusinessException for the single-send path
        with pytest.raises(BusinessException) as exc:
            await producer.send(events[0])
        assert exc.value.code == BusinessErrorCode.EventTooLarge

        # send_batch should drop the event silently and call send_batch with []
        result = await producer.send_batch(events)
        assert result.successful_ids == []
        assert len(client.batches) == 1
        assert client.batches[0] == []
    finally:
        EventProducer._MAX_PROTO_BYTES = original_limit
