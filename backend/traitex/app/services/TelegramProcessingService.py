import datetime
import logging
from datetime import timezone
from typing import List
from uuid import UUID

from app.domain.exceptions.processing.DuplicateProcessingInfoException import DuplicateProcessingInfoException
from app.domain.interfaces.business_logic.ITelegramProcessingService import ITelegramProcessingService
from app.domain.interfaces.messaging.IEventProducer import IEventProducer
from app.domain.models.ConnectorTypeModel import ConnectorTypeModel
from app.domain.models.events.enriched.EnrichedEventModel import EnrichedEventModel
from app.domain.models.events.raw.telegram.RawTelegramMessage import RawTelegramMessage
from app.domain.models.users.ProcessedUserModel import ProcessedUserModel
from app.domain.models.users.UserForProcessingModel import UserForProcessingModel
from app.domain.models.users.processingCredentials.TelegramProcessingCredentialsModel import \
    TelegramProcessingCredentialsModel
from app.services.segmentation import (
    BufferedMessage,
    ConversationSegment,
    SegmentBuffer,
    SegmentationConfig,
    build_segment_event,
    collapse_albums,
)
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
            segment_buffer: SegmentBuffer | None = None,
    ):
        self.telegram_processing_repository = telegram_processing_repository
        self.processing_results_repository = processing_results_repository
        self.processing_snapshot_repository = processing_snapshot_repository
        self.state_tracking_client = state_tracking_client
        self.telegram_api_client = telegram_api_client
        self.event_producer = event_producer
        self._segment_buffer = segment_buffer or SegmentBuffer(SegmentationConfig())
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

        active_chat_ids_by_user: dict[UUID, frozenset[int]] = {}
        for user in users_for_processing:
            credentials = user.credentials
            if isinstance(credentials, TelegramProcessingCredentialsModel):
                active_chat_ids_by_user[user.user_id] = credentials.active_chat_ids
            else:
                active_chat_ids_by_user[user.user_id] = frozenset()

        users_to_process = [
            u for u in users_for_processing
            if active_chat_ids_by_user.get(u.user_id)
        ]
        skipped_users = [
            u for u in users_for_processing
            if not active_chat_ids_by_user.get(u.user_id)
        ]
        if skipped_users:
            self.logger.debug(
                f"Skipping {len(skipped_users)} Telegram users with no active chats"
            )
            now = datetime.datetime.now(timezone.utc)
            await self.state_tracking_client.mark_processed_users(
                [
                    ProcessedUserModel(user_id=u.user_id, processed_at=now)
                    for u in skipped_users
                ],
                service_type=ConnectorTypeModel.Telegram,
            )
        if not users_to_process:
            return

        processing_info = await self.telegram_processing_repository.get_users_processing_info(
            user_ids=[u.user_id for u in users_to_process]
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
        for user in users_to_process:
            last_id = last_processing_map.get(user.user_id)
            users_with_last_ids.append((user, last_id))

        should_mark_processed_with_retained = await self.processing_snapshot_repository.belongs_to_snapshot(
            datetime.datetime.now(timezone.utc)
        )

        fetch_results = await self.telegram_api_client.fetch_batch_messages(users_with_last_ids)

        await self._process_fetch_results(
            fetch_results,
            retain_processed_messages=should_mark_processed_with_retained,
            active_chat_ids_by_user=active_chat_ids_by_user,
        )

    async def flush_stale_segments(self) -> None:
        """Emit segments whose silence window elapsed even if no new messages
        arrived for the user this polling cycle. Called by the consumer on
        every tick to bound flush latency.
        """
        now = datetime.datetime.now(timezone.utc)
        stale = self._segment_buffer.flush_stale(now)
        if not stale:
            return

        retain = await self.processing_snapshot_repository.belongs_to_snapshot(now)
        for segment in stale:
            await self._emit_segment(segment, retain_processed_messages=retain)

    async def _process_fetch_results(
            self,
            results: dict[UUID, UserTelegramFetchResult],
            retain_processed_messages: bool,
            active_chat_ids_by_user: dict[UUID, frozenset[int]],
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

            await self._process_user_messages(
                user_id,
                result.messages,
                retain_processed_messages,
                active_chat_ids=active_chat_ids_by_user.get(user_id, frozenset()),
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
            retain_processed_messages: bool,
            active_chat_ids: frozenset[int],
    ) -> None:
        """Group messages into conversation segments and emit closed ones."""
        self.logger.info(
            f"Processing {len(messages)} Telegram messages for user {user_id} "
            f"through segmentation buffer"
        )

        segments_to_emit: list[ConversationSegment] = []

        # Sort by chat then by message_id so albums sit next to each other —
        # collapse_albums tolerates any ordering, but predictable order keeps
        # logs and tests readable.
        messages = sorted(messages, key=lambda m: (m.chat_id, m.message_id))
        by_chat: dict[int, list[RawTelegramMessage]] = {}
        for m in messages:
            by_chat.setdefault(m.chat_id, []).append(m)

        for chat_id, chat_messages in by_chat.items():
            if chat_id not in active_chat_ids:
                continue
            sample = chat_messages[0]
            chat_title = sample.chat_title
            chat_type = sample.chat_type

            buffered = collapse_albums(chat_messages)

            if chat_type == "channel":
                # Per-post emission for broadcast channels: each (album-collapsed)
                # message becomes its own one-message segment. We bypass the
                # silence buffer entirely so news posts surface immediately.
                for bm in buffered:
                    if bm.is_noise:
                        continue
                    segments_to_emit.append(
                        self._wrap_single_message(user_id, sample, bm)
                    )
                continue

            for bm in buffered:
                _, overflow = self._segment_buffer.add(
                    user_id=user_id,
                    chat_id=chat_id,
                    chat_title=chat_title,
                    chat_type=chat_type,
                    message=bm,
                )
                if overflow is not None:
                    segments_to_emit.append(overflow)

        # After ingesting this batch, also drain segments whose silence window
        # has lapsed (e.g. an idle chat with no new messages this tick is
        # handled by the periodic consumer hook, but a quiet chat that just
        # received its final message should close right away).
        now = datetime.datetime.now(timezone.utc)
        segments_to_emit.extend(self._segment_buffer.flush_stale(now))

        for segment in segments_to_emit:
            await self._emit_segment(segment, retain_processed_messages=retain_processed_messages)

    def _wrap_single_message(
            self,
            user_id: UUID,
            sample: RawTelegramMessage,
            buffered: BufferedMessage,
    ) -> ConversationSegment:
        seg = ConversationSegment(
            user_id=user_id,
            chat_id=sample.chat_id,
            chat_title=sample.chat_title,
            chat_type=sample.chat_type,
        )
        seg.add(buffered)
        return seg

    async def _emit_segment(
            self,
            segment: ConversationSegment,
            retain_processed_messages: bool,
    ) -> None:
        event: EnrichedEventModel | None = build_segment_event(segment)
        if event is None:
            return

        processed_at = datetime.datetime.now(timezone.utc)

        if retain_processed_messages:
            await self.processing_results_repository.save_processing_result(
                processed_at,
                event,
            )

        await self.event_producer.send(event)
