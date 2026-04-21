import logging
import uuid
import datetime
from datetime import timezone
from uuid import UUID

from app.domain.exceptions.processing.DuplicateProcessingInfoException import DuplicateProcessingInfoException
from app.domain.interfaces.business_logic.IGmailProcessingService import IGmailProcessingService
from app.domain.interfaces.business_logic.IGoogleCalendarProcessingService import IGoogleCalendarProcessingService
from app.domain.interfaces.messaging.IEventProducer import IEventProducer
from app.domain.models.ConnectorTypeModel import ConnectorTypeModel
from app.domain.models.events.enriched.EnrichedEventModel import EnrichedEventModel
from app.domain.models.events.raw.gmail.RawGmailMessage import RawGmailMessage
from app.domain.models.traits.AttachmentTrait import AttachmentTrait, Attachment
from app.domain.models.traits.RecipientTrait import RecipientTrait
from app.domain.models.traits.SenderTrait import SenderTrait
from app.domain.models.traits.SubjectTrait import SubjectTrait
from app.domain.models.traits.base.TraitModel import TraitModel
from app.domain.models.traits.common.UserIdentifier import UserIdentifier
from app.domain.models.users.ProcessedUserModel import ProcessedUserModel
from app.domain.models.users.UserForProcessingModel import UserForProcessingModel
from dataAccess.interfaces.IGmailProcessingRepository import IGmailProcessingRepository
from dataAccess.interfaces.IProcessingResultsRepository import IProcessingResultsRepository
from dataAccess.interfaces.IProcessingSnapshotRepository import IProcessingSnapshotRepository
from dataAccess.models.gmail.UserGmailProcessingInfo import UserGmailProcessingInfo
from externalClients.gmail_api.GmailApiClient import GmailApiClient
from externalClients.gmail_api.models.UserGmailFetchResult import UserGmailFetchResult
from externalClients.personage_auth.StateTrackingClient import StateTrackingClient


class GoogleCalendarProcessingService(IGoogleCalendarProcessingService):
    USERS_FOR_PROCESSING_BATCH_SIZE = 10
    SECONDS_SINCE_LAST_PROCESS = 1 * 60

    def __init__(
            self,
            google_calendar_processing_repository: IGoogleCalendarProcessingRepository,
            processing_results_repository: IProcessingResultsRepository,
            processing_snapshot_repository: IProcessingSnapshotRepository,
            state_tracking_client: StateTrackingClient,
            google_calendar_api_client: GoogleCalendarApiClient,
            event_producer: IEventProducer,
    ):
        self.google_calendar_processing_repository = google_calendar_processing_repository
        self.processing_results_repository = processing_results_repository
        self.processing_snapshot_repository = processing_snapshot_repository
        self.state_tracking_client = state_tracking_client
        self.google_calendar_api_client = google_calendar_api_client
        self.event_producer = event_producer
        self.logger = logging.getLogger("[GoogleCalendarProcessingService]")


    async def get_users_for_processing(self) -> list[UserForProcessingModel]:
        return await self.state_tracking_client.get_users_for_processing(
            batch_size=GoogleCalendarProcessingService.USERS_FOR_PROCESSING_BATCH_SIZE,
            seconds_since_last_process=GoogleCalendarProcessingService.SECONDS_SINCE_LAST_PROCESS,
            service_type=ConnectorTypeModel.Gmail,
        )

    async def process_users_events(self, users_for_processing: list[UserForProcessingModel]) -> None:
        processing_info = await self.gmail_processing_repository.get_users_processing_info(
            user_ids=[u.user_id for u in users_for_processing]
        )

        last_processing_map = {}
        for user in processing_info:
            if user.user_id in last_processing_map:
                raise DuplicateProcessingInfoException(
                    user.user_id,
                    connector_type=ConnectorTypeModel.Gmail,
                    processing_info_source="PostgreSQL gmail_processing table"
                )

            last_processing_map[user.user_id] = user.last_message_history_id

        users_with_last_ids = []
        for user in users_for_processing:
            last_id = last_processing_map.get(user.user_id)
            users_with_last_ids.append((user, last_id))

        should_mark_processed_with_retained = await self.processing_snapshot_repository.belongs_to_snapshot(datetime.datetime.now(timezone.utc))

        fetch_results = await self.gmail_api_client.fetch_batch_messages(users_with_last_ids)
        await self._process_fetch_results(fetch_results, retain_processed_messages=should_mark_processed_with_retained)

    async def _process_fetch_results(self, results: dict[UUID, UserGmailFetchResult], retain_processed_messages: bool) -> None:
        successful_fetches = [
            UserGmailProcessingInfo(user_id=user_id, last_message_history_id=fetch.new_history_id)
            for user_id, fetch in results.items()
            if fetch.success and fetch.new_history_id
        ]

        user_processed_at_map: dict[UUID, datetime.datetime] = {}
        for user_id, result in results.items():
            if not result.success:
                self.logger.error(f"Failed to fetch messages for user {user_id}: {result.error_message}")
                continue

            if not result.new_history_id:
                self.logger.warning(f"No new history id found for user {user_id}")
                continue

            if result.messages:
                await self._process_user_messages(user_id, result.messages, retain_processed_messages)
                user_processed_at_map[user_id] = datetime.datetime.now(datetime.UTC)

        await self.gmail_processing_repository.save_users_processing_info(successful_fetches)
        current_timestamp = datetime.datetime.now(datetime.UTC)
        await self.state_tracking_client.mark_processed_users(
            [ProcessedUserModel(
                user_id=user_id,
                processed_at=user_processed_at_map[user_id] if user_id in user_processed_at_map else current_timestamp,
            ) for user_id, _ in results.items()],
            service_type=ConnectorTypeModel.Gmail,
        )

    async def _process_user_messages(
            self,
            user_id: UUID,
            messages: list[RawGmailMessage],
            retain_processed_messages: bool
    ) -> None:
        self. logger.info(f"Processing {len(messages)} messages for user {user_id}")

        for message in messages:
            enriched_message = GmailProcessingService._enrich_message(user_id, message)

            processed_at = datetime.datetime.now(datetime.UTC)
            #TODO: consider batch write
            if retain_processed_messages:
                await self.processing_results_repository.save_processing_result(processed_at, enriched_message)
            await self.event_producer.send(enriched_message)

    @staticmethod
    def _enrich_message(user_id: UUID, raw_message: RawGmailMessage) -> EnrichedEventModel:
        traits: list[TraitModel] = [
            SubjectTrait(name=raw_message.subject),
            RecipientTrait(recipients=[UserIdentifier(email=r.email) for r in raw_message.to_emails]),
            AttachmentTrait(attachments=[Attachment(name=x.filename) for x in raw_message.attachments]),
            SenderTrait(identifier=UserIdentifier(email=raw_message.from_email.email)),
        ]

        return EnrichedEventModel(
            id=uuid.uuid4(),
            user_id=user_id,
            connector_type=ConnectorTypeModel.Gmail,
            occurred_at=raw_message.received_date,
            main_body=raw_message.body,
            traits=traits,
        )
