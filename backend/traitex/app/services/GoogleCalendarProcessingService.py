import logging
import uuid
import datetime
from datetime import timezone
from uuid import UUID
from app.domain.exceptions.processing.DuplicateProcessingInfoException import DuplicateProcessingInfoException
from app.domain.interfaces.business_logic.ICalendarProcessingService import ICalendarProcessingService
from app.domain.interfaces.messaging.IEventProducer import IEventProducer
from app.domain.models.ConnectorTypeModel import ConnectorTypeModel
from app.domain.models.events.raw.calendar.RawCalendarEvent import RawCalendarEvent

from app.domain.models.events.enriched.EnrichedEventModel import EnrichedEventModel
from app.domain.models.traits.AttachmentTrait import AttachmentTrait, Attachment
from app.domain.models.traits.RecipientTrait import RecipientTrait
from app.domain.models.traits.SenderTrait import SenderTrait
from app.domain.models.traits.SubjectTrait import SubjectTrait
from app.domain.models.traits.TraitUnion import TraitUnion
from app.domain.models.traits.common.UserIdentifier import UserIdentifier
from app.domain.models.users.ProcessedUserModel import ProcessedUserModel
from app.domain.models.users.UserForProcessingModel import UserForProcessingModel
from dataAccess.interfaces.ICalendarProcessingRepository import ICalendarProcessingRepository
from dataAccess.interfaces.IProcessingResultsRepository import IProcessingResultsRepository
from dataAccess.interfaces.IProcessingSnapshotRepository import IProcessingSnapshotRepository
from dataAccess.models.googleCalendar.UserCalendarProcessingInfo import UserCalendarProcessingInfo
from externalClients.calendar_api.CalendarApiClient import CalendarApiClient
from externalClients.calendar_api.models.UserCalendarFetchResult import UserCalendarFetchResult
from externalClients.personage_auth.StateTrackingClient import StateTrackingClient


class CalendarProcessingService(ICalendarProcessingService):
    USERS_FOR_PROCESSING_BATCH_SIZE = 10
    SECONDS_SINCE_LAST_PROCESS = 1 * 60  # 1 minute

    def __init__(
            self,
            calendar_processing_repository: ICalendarProcessingRepository,
            processing_results_repository: IProcessingResultsRepository,
            processing_snapshot_repository: IProcessingSnapshotRepository,
            state_tracking_client: StateTrackingClient,
            calendar_api_client: CalendarApiClient,
            event_producer: IEventProducer,
    ):
        self.calendar_processing_repository = calendar_processing_repository
        self.processing_results_repository = processing_results_repository
        self.processing_snapshot_repository = processing_snapshot_repository
        self.state_tracking_client = state_tracking_client
        self.calendar_api_client = calendar_api_client
        self.event_producer = event_producer
        self.logger = logging.getLogger("[CalendarProcessingService]")

    async def get_users_for_processing(self) -> list[UserForProcessingModel]:
        return await self.state_tracking_client.get_users_for_processing(
            batch_size=CalendarProcessingService.USERS_FOR_PROCESSING_BATCH_SIZE,
            seconds_since_last_process=CalendarProcessingService.SECONDS_SINCE_LAST_PROCESS,
            service_type=ConnectorTypeModel.GoogleCalendar,
        )

    async def process_users_events(self, users_for_processing: list[UserForProcessingModel]) -> None:
        processing_info = await self.calendar_processing_repository.get_users_processing_info(
            user_ids=[u.user_id for u in users_for_processing]
        )

        # Build map of user_id -> sync_token
        last_sync_map = {}
        for user_info in processing_info:
            if user_info.user_id in last_sync_map:
                raise DuplicateProcessingInfoException(
                    user_info.user_id,
                    connector_type=ConnectorTypeModel.GoogleCalendar,
                    processing_info_source="PostgreSQL calendar_processing table"
                )
            last_sync_map[user_info.user_id] = user_info.last_sync_token

        users_with_sync_tokens = []
        for user in users_for_processing:
            sync_token = last_sync_map.get(user.user_id)
            users_with_sync_tokens.append((user, sync_token))

        should_mark_processed_with_retained = await self.processing_snapshot_repository.belongs_to_snapshot(
            datetime.datetime.now(timezone.utc)
        )

        fetch_results = await self.calendar_api_client.fetch_batch_events(users_with_sync_tokens)

        await self._process_fetch_results(fetch_results, retain_processed_events=should_mark_processed_with_retained)

    async def _process_fetch_results(
            self,
            results: dict[UUID, UserCalendarFetchResult],
            retain_processed_events: bool
    ) -> None:
        successful_fetches = []
        for user_id, fetch in results.items():
            if fetch.success and fetch.new_sync_token:
                successful_fetches.append(
                    UserCalendarProcessingInfo(
                        user_id=user_id,
                        last_sync_token=fetch.new_sync_token,
                        last_event_updated_time=datetime.datetime.now(timezone.utc)
                    )
                )

        user_processed_at_map: dict[UUID, datetime.datetime] = {}

        for user_id, result in results.items():
            if not result.success:
                self.logger.error(f"Failed to fetch events for user {user_id}: {result.error_message}")
                continue

            if result.events:
                await self._process_user_events(user_id, result.events, retain_processed_events)
                user_processed_at_map[user_id] = datetime.datetime.now(datetime.UTC)
            else:
                self.logger.debug(f"No new events found for user {user_id}")

        if successful_fetches:
            await self.calendar_processing_repository.save_users_processing_info(successful_fetches)

        current_timestamp = datetime.datetime.now(datetime.UTC)
        await self.state_tracking_client.mark_processed_users(
            [
                ProcessedUserModel(
                    user_id=user_id,
                    processed_at=user_processed_at_map.get(user_id, current_timestamp),
                )
                for user_id in results.keys()
            ],
            service_type=ConnectorTypeModel.GoogleCalendar,
        )

    async def _process_user_events(
            self,
            user_id: UUID,
            events: list[RawCalendarEvent],
            retain_processed_events: bool
    ) -> None:
        self.logger.info(f"Processing {len(events)} events for user {user_id}")

        active_events = [e for e in events if e.status != 'cancelled']
        cancelled_events = [e for e in events if e.status == 'cancelled']

        if cancelled_events:
            self.logger.info(f"  {len(cancelled_events)} cancelled/deleted events for user {user_id}")

        for event in active_events:
            enriched_event = self._enrich_event(user_id, event)
            processed_at = datetime.datetime.now(datetime.UTC)

            if retain_processed_events:
                await self.processing_results_repository.save_processing_result(
                    processed_at,
                    enriched_event
                )

            await self.event_producer.send(enriched_event)

        for cancelled_event in cancelled_events:
            await self._handle_cancelled_event(user_id, cancelled_event, retain_processed_events)

    async def _handle_cancelled_event(
            self,
            user_id: UUID,
            cancelled_event: RawCalendarEvent,
            retain_processed_events: bool
    ) -> None:
        enriched_event = self._enrich_event(user_id, cancelled_event)

        if retain_processed_events:
            processed_at = datetime.datetime.now(datetime.UTC)
            await self.processing_results_repository.save_processing_result(
                processed_at,
                enriched_event
            )

        await self.event_producer.send(enriched_event)

        self.logger.debug(f"Sent cancellation event for {cancelled_event.id} to user {user_id}")

    @staticmethod
    def _enrich_event(user_id: UUID, raw_event: RawCalendarEvent) -> EnrichedEventModel:
        traits: list[TraitUnion] = []

        if raw_event.summary:
            traits.append(SubjectTrait(name=raw_event.summary))

        if raw_event.organizer:
            traits.append(SenderTrait(
                identifier=UserIdentifier(
                    email=raw_event.organizer.email,
                    name=raw_event.organizer.display_name
                )
            ))

        if raw_event.attendees:
            recipients = []
            for attendee in raw_event.attendees:
                if attendee.email:
                    recipients.append(UserIdentifier(
                        email=attendee.email,
                        name=attendee.display_name
                    ))
            if recipients:
                traits.append(RecipientTrait(recipients=recipients))

        if raw_event.attachments:
            traits.append(AttachmentTrait(
                attachments=[Attachment(name=att.filename) for att in raw_event.attachments]
            ))

        occurred_at = raw_event.start_time if raw_event.start_time else raw_event.created_time

        return EnrichedEventModel(
            id=uuid.uuid4(),
            user_id=user_id,
            connector_type=ConnectorTypeModel.GoogleCalendar,
            occurred_at=occurred_at,
            main_body=raw_event.description or "",
            traits=traits,
        )