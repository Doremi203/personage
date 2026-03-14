import datetime
import logging
import uuid
from datetime import timezone
from typing import List
from uuid import UUID

from app.domain.exceptions.processing.DuplicateProcessingInfoException import DuplicateProcessingInfoException
from app.domain.interfaces.business_logic.ITelegramProcessingService import ITelegramProcessingService
from app.domain.interfaces.messaging.IEventProducer import IEventProducer
from app.domain.models.ConnectorTypeModel import ConnectorTypeModel
from app.domain.models.events.enriched.EnrichedEventModel import EnrichedEventModel
from app.domain.models.events.raw.telegram.RawTelegramMessage import RawTelegramMessage
from app.domain.models.traits.SenderTrait import SenderTrait
from app.domain.models.traits.base.TraitModel import TraitModel
from app.domain.models.traits.common.UserIdentifier import UserIdentifier
from app.domain.models.users.ProcessedUserModel import ProcessedUserModel
from app.domain.models.users.UserForProcessingModel import UserForProcessingModel
from dataAccess.interfaces.IProcessingResultsRepository import IProcessingResultsRepository
from dataAccess.interfaces.IProcessingSnapshotRepository import IProcessingSnapshotRepository
from dataAccess.interfaces.ITelegramProcessingRepository import ITelegramProcessingRepository
from dataAccess.models.telegram.UserTelegramProcessingInfo import UserTelegramProcessingInfo
from externalClients.personage_auth.StateTrackingClient import StateTrackingClient
from externalClients.telegram.TelegramApiClient import TelegramApiClient
from externalClients.telegram.models.UserTelegramFetchResult import UserTelegramFetchResult


class TelegramProcessingService(ITelegramProcessingService):
    USERS_FOR_PROCESSING_BATCH_SIZE = 5
    SECONDS_SINCE_LAST_PROCESS = 5 * 60

    def __init__(
            self,
            telegram_processing_repository: ITelegramProcessingRepository,
            processing_results_repository: IProcessingResultsRepository,
            processing_snapshot_repository: IProcessingSnapshotRepository,
            state_tracking_client: StateTrackingClient,
            telegram_api_client: TelegramApiClient,
            event_producer: IEventProducer,
    ):
        self.telegram_processing_repository = telegram_processing_repository
        self.processing_results_repository = processing_results_repository
        self.processing_snapshot_repository = processing_snapshot_repository
        self.state_tracking_client = state_tracking_client
        self.telegram_api_client = telegram_api_client
        self.event_producer = event_producer
        self.logger = logging.getLogger("[TelegramProcessingService]")

    async def get_users_for_processing(self) -> List[UserForProcessingModel]:
        """Get users who need Telegram message processing"""
        return await self.state_tracking_client.get_users_for_processing(
            batch_size=TelegramProcessingService.USERS_FOR_PROCESSING_BATCH_SIZE,
            seconds_since_last_process=TelegramProcessingService.SECONDS_SINCE_LAST_PROCESS,
            service_type=ConnectorTypeModel.Telegram,
        )

    async def process_users_events(self, users_for_processing: List[UserForProcessingModel]) -> None:
        """Process Telegram messages for the given users"""
        if not users_for_processing:
            return

        processing_info = await self.telegram_processing_repository.get_users_processing_info(
            user_ids=[u.user_id for u in users_for_processing]
        )

        last_processing_map = {}
        for info in processing_info:
            if info.user_id in last_processing_map:
                raise DuplicateProcessingInfoException(
                    info.user_id,
                    connector_type=ConnectorTypeModel.Telegram,
                    processing_info_source="PostgreSQL telegram_processing table"
                )
            last_processing_map[info.user_id] = info.last_message_id

        users_with_last_ids = []
        for user in users_for_processing:
            last_id = last_processing_map.get(user.user_id)
            users_with_last_ids.append((user, last_id))

        should_mark_processed_with_retained = await self.processing_snapshot_repository.belongs_to_snapshot(
            datetime.datetime.now(timezone.utc)
        )

        fetch_results = await self.telegram_api_client.fetch_batch_messages(users_with_last_ids)

        await self._process_fetch_results(
            fetch_results,
            retain_processed_messages=should_mark_processed_with_retained
        )

    async def _process_fetch_results(
            self,
            results: dict[UUID, UserTelegramFetchResult],
            retain_processed_messages: bool
    ) -> None:
        """Process fetch results and update state"""
        successful_fetches = []
        user_processed_at_map = {}

        for user_id, result in results.items():
            if not result.success:
                self.logger.error(f"Failed to fetch Telegram messages for user {user_id}: {result.error_message}")
                continue

            if not result.messages:
                self.logger.info(f"No new Telegram messages for user {user_id}")
                user_processed_at_map[user_id] = datetime.datetime.now(timezone.utc)
                continue

            if result.messages:
                await self._process_user_messages(
                    user_id,
                    result.messages,
                    retain_processed_messages
                )
                user_processed_at_map[user_id] = datetime.datetime.now(timezone.utc)

            if result.new_last_message_id:
                successful_fetches.append(
                    UserTelegramProcessingInfo(
                        user_id=user_id,
                        last_message_id=result.new_last_message_id
                    )
                )

        if successful_fetches:
            await self.telegram_processing_repository.save_users_processing_info(successful_fetches)

        current_timestamp = datetime.datetime.now(timezone.utc)
        processed_users = [
            ProcessedUserModel(
                user_id=user_id,
                processed_at=user_processed_at_map.get(user_id, current_timestamp)
            )
            for user_id in results.keys()
        ]

        if processed_users:
            await self.state_tracking_client.mark_processed_users(
                processed_users,
                service_type=ConnectorTypeModel.Telegram
            )

    async def _process_user_messages(
            self,
            user_id: UUID,
            messages: List[RawTelegramMessage],
            retain_processed_messages: bool
    ) -> None:
        """Process and enrich messages for a user"""
        self.logger.info(f"Processing {len(messages)} Telegram messages for user {user_id}")

        for message in messages:
            enriched_message = self._enrich_message(user_id, message)

            processed_at = datetime.datetime.now(timezone.utc)

            if retain_processed_messages:
                await self.processing_results_repository.save_processing_result(
                    processed_at,
                    enriched_message
                )

            await self.event_producer.send(enriched_message)

    @staticmethod
    def _enrich_message(user_id: UUID, raw_message: RawTelegramMessage) -> EnrichedEventModel:
        """Enrich raw Telegram message with traits"""
        traits: List[TraitModel] = [
            SenderTrait(
                identifier=UserIdentifier(
                    telegram_id=str(raw_message.sender_id) if raw_message.sender_id else None,
                    telegram_tag=raw_message.sender_username,
                    telegram_name=f"{raw_message.sender_first_name or ''} {raw_message.sender_last_name or ''}".strip()
                )
            )
        ]

        return EnrichedEventModel(
            id=uuid.uuid4(),
            user_id=user_id,
            connector_type=ConnectorTypeModel.Telegram,
            occurred_at=raw_message.date,
            main_body=raw_message.text,
            traits=traits,
        )