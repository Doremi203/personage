import base64
import hashlib
import logging
from collections.abc import Iterable
from typing import Any
from uuid import UUID

from google.protobuf.timestamp_pb2 import Timestamp

from app.domain.exceptions.base.BusinessErrorCode import BusinessErrorCode
from app.domain.exceptions.base.BusinessException import BusinessException
from app.domain.exceptions.messaging.DuplicateTraitEncountered import DuplicateTraitEncountered
from app.domain.interfaces.messaging.IEventProducer import IEventProducer
from app.domain.interfaces.messaging.IQueueClient import (
    BatchResult,
    IQueueClient,
    QueueEntry,
)
from app.domain.models.ConnectorTypeModel import ConnectorTypeModel
from app.domain.models.events.enriched.EnrichedEventModel import EnrichedEventModel
from app.domain.models.traits.AttachmentTrait import AttachmentTrait
from app.domain.models.traits.RecipientTrait import RecipientTrait
from app.domain.models.traits.SenderTrait import SenderTrait
from app.domain.models.traits.SubjectTrait import SubjectTrait
from app.domain.models.traits.base.TraitKindModel import TraitKindModel
from app.domain.models.traits.base.TraitModel import TraitModel
from proto.events_pb2 import ConnectorType
from proto import events_pb2

logger = logging.getLogger(__name__)


class EventProducer(IEventProducer):
    MAX_BATCH_SIZE = 10

    # YMQ/SQS hard limit: 256 KiB total (body + attributes + framing).
    # The message body is base64-encoded protobuf, which inflates by 4/3.
    # We reserve 4 KiB for attributes/framing, then divide by the base64
    # ratio to get the maximum protobuf payload size (~189 KiB).
    _YMQ_MAX_MESSAGE_BYTES = 256 * 1024
    _ATTRIBUTE_BUDGET = 4 * 1024
    _MAX_PROTO_BYTES = int((_YMQ_MAX_MESSAGE_BYTES - _ATTRIBUTE_BUDGET) * 3 / 4)
    _TRUNCATION_MARKER = "\n\n[... truncated by traitex ...]"
    _SIZE_SLACK = 32

    def __init__(
            self,
            queue_client: IQueueClient
    ):
        self.client = queue_client

    async def send(
            self,
            event: EnrichedEventModel,
            additional_attributes: dict[str, Any] | None = None,
            target_queue_url: str | None = None
    ) -> dict[str, Any]:
        entry = self._build_entry(event, additional_attributes)
        return await self.client.send_single(
            deduplication_id=entry.deduplication_id,
            message_group_id=entry.message_group_id,
            message_body=entry.message_body,
            message_attributes=entry.message_attributes,
            target_queue_url=target_queue_url,
        )

    async def send_batch(
            self,
            events: Iterable[EnrichedEventModel],
            additional_attributes: dict[str, Any] | None = None,
            target_queue_url: str | None = None
    ) -> BatchResult:
        entries: list[QueueEntry] = []
        skipped: list[BusinessException] = []
        for event in events:
            try:
                entries.append(self._build_entry(event, additional_attributes))
            except BusinessException as exc:
                skipped.append(exc)
                logger.warning(
                    "Dropping event %s before batch send: %s",
                    event.id,
                    exc,
                )

        result = await self.client.send_batch(entries, target_queue_url=target_queue_url)
        if skipped:
            logger.warning(
                "send_batch dropped %s events before sending due to size or build errors",
                len(skipped),
            )
        return result

    def _build_entry(
            self,
            event: EnrichedEventModel,
            additional_attributes: dict[str, Any] | None,
    ) -> QueueEntry:
        proto_event = self._flatten_event(event)
        proto_event = self._enforce_size_limit(proto_event, event.id)

        binary_data = proto_event.SerializeToString(deterministic=True)
        message_body = base64.b64encode(binary_data).decode('utf-8')

        message_attributes: dict[str, Any] = {
            'EventType': 'api.events.Event',
            'Connector': event.connector_type.name,
            'UserId': str(event.user_id),
            'MessageFormat': 'protobuf/base64',
        }
        if additional_attributes:
            message_attributes.update(additional_attributes)

        return QueueEntry(
            id=str(event.id)[:80],
            deduplication_id=self._compute_dedup_id(proto_event),
            message_group_id=str(event.user_id),
            message_body=message_body,
            message_attributes=message_attributes,
        )

    @classmethod
    def _compute_dedup_id(cls, proto_event: events_pb2.Event) -> str:
        clone = events_pb2.Event()
        clone.CopyFrom(proto_event)
        clone.ClearField("id")
        digest = hashlib.sha256(clone.SerializeToString(deterministic=True)).hexdigest()
        return digest

    @classmethod
    def _enforce_size_limit(
            cls,
            proto_event: events_pb2.Event,
            event_id: UUID,
    ) -> events_pb2.Event:
        serialized = proto_event.SerializeToString()
        original_size = len(serialized)
        if original_size <= cls._MAX_PROTO_BYTES:
            return proto_event

        body_bytes = proto_event.context.body.encode('utf-8')
        marker_bytes = cls._TRUNCATION_MARKER.encode('utf-8')
        overflow = original_size - cls._MAX_PROTO_BYTES
        keep_bytes = len(body_bytes) - overflow - len(marker_bytes) - cls._SIZE_SLACK

        if keep_bytes > 0:
            truncated = body_bytes[:keep_bytes].decode('utf-8', errors='ignore')
            proto_event.context.body = truncated + cls._TRUNCATION_MARKER
        else:
            proto_event.context.body = cls._TRUNCATION_MARKER

        if len(proto_event.SerializeToString()) > cls._MAX_PROTO_BYTES:
            del proto_event.context.attachments[:]
        if len(proto_event.SerializeToString()) > cls._MAX_PROTO_BYTES:
            del proto_event.context.other_participants[:]
        if len(proto_event.SerializeToString()) > cls._MAX_PROTO_BYTES:
            proto_event.context.body = cls._TRUNCATION_MARKER

        final_size = len(proto_event.SerializeToString())
        if final_size > cls._MAX_PROTO_BYTES:
            raise BusinessException(
                BusinessErrorCode.EventTooLarge,
                f"Event {event_id} cannot fit YMQ size limit even after truncation "
                f"(final={final_size} bytes, max={cls._MAX_PROTO_BYTES} bytes)",
            )

        logger.warning(
            "Truncated event %s (connector=%s) from %s to %s proto bytes to fit YMQ size limit",
            event_id,
            ConnectorType.Name(proto_event.connector_type),
            original_size,
            final_size,
        )
        return proto_event

    @staticmethod
    def _flatten_event(event: EnrichedEventModel) -> events_pb2.Event:
        proto_event = events_pb2.Event()

        proto_event.id = str(event.id)
        proto_event.user_id = str(event.user_id)
        proto_event.connector_type = EventProducer._to_grpc_connector_type(event.connector_type)

        timestamp = Timestamp()
        timestamp.FromDatetime(event.occurred_at)
        proto_event.occurred_at.CopyFrom(timestamp)

        context = EventProducer._extract_context(event)
        proto_event.context.CopyFrom(context)

        return proto_event

    @staticmethod
    def _extract_context(event: EnrichedEventModel) -> events_pb2.Context:

        EventProducer._throw_on_duplicate_trait(event.traits)
        context = events_pb2.Context()

        context.body = event.main_body

        for trait in event.traits:
            match trait:
                case SubjectTrait() as subjectTrait:
                    context.subject.name = subjectTrait.name
                case AttachmentTrait() as attachmentTrait:
                    context.attachments.extend([events_pb2.Context.Attachment(name=attachment.name)
                                                for attachment in attachmentTrait.attachments])
                case RecipientTrait() as recipientTrait:
                    context.other_participants.extend([events_pb2.Context.Participant(email=recipient.email)
                                                       for recipient in recipientTrait.recipients])
                case SenderTrait() as senderTrait:
                    if senderTrait.identifier.email:
                        context.sender.email = senderTrait.identifier.email
                    if senderTrait.identifier.telegram_id:
                        telegram_user = events_pb2.TelegramUser(
                            id=senderTrait.identifier.telegram_id
                        )

                        if senderTrait.identifier.telegram_tag:
                            telegram_user.tag = senderTrait.identifier.telegram_tag

                        if senderTrait.identifier.telegram_name:
                            telegram_user.name = senderTrait.identifier.telegram_name

                        context.sender.telegram_user.CopyFrom(telegram_user)
                case _:
                    raise Exception(f"Unknown trait type {type(trait)}")

        return context

    @staticmethod
    def _throw_on_duplicate_trait(traits: list[TraitModel]) -> None:
        trait_set: set[TraitKindModel] = set()
        for trait in traits:
            if trait.kind in trait_set:
                raise DuplicateTraitEncountered(trait.kind)
            trait_set.add(trait.kind)

    @staticmethod
    def _to_grpc_connector_type(connector_type: ConnectorTypeModel) -> ConnectorType.ValueType:
        match connector_type:
            case ConnectorTypeModel.Gmail:
                return ConnectorType.CONNECTOR_TYPE_GMAIL
        return ConnectorType.CONNECTOR_TYPE_UNKNOWN
