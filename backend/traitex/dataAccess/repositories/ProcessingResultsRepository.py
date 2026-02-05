from datetime import datetime

from pydapper.commands import CommandsAsync

from app.domain.models.events.enriched.EnrichedEventModel import EnrichedEventModel
from dataAccess.infrastructure.IPgConnectionProvider import IPgConnectionProvider
from dataAccess.interfaces.IProcessingResultsRepository import IProcessingResultsRepository


class ProcessingResultsRepository(IProcessingResultsRepository):
    def __init__(
            self,
            connection_provider: IPgConnectionProvider
    ):
        self.connection_provider = connection_provider

    async def save_processing_result(
            self,
            processed_at: datetime,
            processing_result: EnrichedEventModel
    ):
        async with self.connection_provider.get_connection() as commands:
            commands: CommandsAsync

            return await commands.execute_async(
                '''
                INSERT INTO processing_result(processed_at, event)
                VALUE(?processed_at?, ?event?);
                ''',
                param={
                    "processed_at": processed_at,
                    "event": processing_result.model_dump_json()
                }
            )

    async def get_processing_results(
            self,
            from_: datetime,
            to: datetime,
            limit: int
    ) -> list[EnrichedEventModel]:
        async with self.connection_provider.get_connection() as commands:
            commands: CommandsAsync
            raw_events = await commands.query_async(
                '''
                SELECT
                    pr.event
                from processing_result pr
                where
                    pr.processed_at between ?from? and ?to?
                order by pr.processed_at asc
                limit ?limit?
                ''',
                param={
                    "from": from_,
                    "to": to,
                    "limit": limit,
                })
            return [EnrichedEventModel.model_validate_json(raw_event['event']) for raw_event in raw_events]
