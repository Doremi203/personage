from uuid import UUID
from app.domain.interfaces.business_logic.IDebugService import IDebugService
from app.domain.models.debug.InternalProcessingInfoModel import InternalProcessingInfoModel
from dataAccess.interfaces.IGmailProcessingRepository import IGmailProcessingRepository


class DebugService(IDebugService):
    def __init__(
            self,
            gmail_processing_repository: IGmailProcessingRepository
    ):
        self.gmail_processing_repository = gmail_processing_repository

    async def rollback_gmail_counter(
            self,
            user_id: UUID,
            decrease_counter_by: int
    ) -> int | None:
        return await self.gmail_processing_repository.decrease_last_history_id(
            user_id=user_id,
            decrease_by=decrease_counter_by
        )

    async def get_processing_info(self) -> list[InternalProcessingInfoModel]:
        gmail_processing_info = await self.gmail_processing_repository.get_all_users_processing_info()
        return [InternalProcessingInfoModel(
            user_id=x.user_id,
            gmail_counter=x.last_message_history_id
        ) for x in gmail_processing_info]
