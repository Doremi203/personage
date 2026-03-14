import base64
from typing import Any

from google.protobuf.timestamp_pb2 import Timestamp

from app.domain.exceptions.messaging.DuplicateTraitEncountered import DuplicateTraitEncountered
from app.domain.interfaces.messaging.IEventProducer import IEventProducer
from app.domain.interfaces.messaging.IQueueClient import IQueueClient
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


class EventProducer(IEventProducer):
    MAX_BATCH_SIZE = 10
    def __init__(
            self,
            queue_client: IQueueClient
    ):
        self.client = queue_client

    async def send(
            self,
            event: EnrichedEventModel,
            additional_attributes: dict[str, Any] | None = None
    ) -> dict[str, Any]:
        proto_event = self._flatten_event(event)
        binary_data = proto_event.SerializeToString()
        message_body = base64.b64encode(binary_data).decode('utf-8')

        message_attributes = {
            'EventType': 'api.events.Event',
            'Connector': event.connector_type.name,
            'UserId': event.user_id,
            'MessageFormat': 'protobuf/base64',
        }

        if additional_attributes:
            message_attributes.update(additional_attributes)

        return await self.client.send_single(
            deduplication_id=str(event.id),
            message_group_id=str(event.user_id),
            message_body=message_body,
            message_attributes=message_attributes
        )

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
