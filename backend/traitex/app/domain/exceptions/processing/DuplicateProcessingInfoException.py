from uuid import UUID

from app.domain.exceptions.base.BusinessErrorCode import BusinessErrorCode
from app.domain.exceptions.base.BusinessException import BusinessException
from app.domain.models.ConnectorTypeModel import ConnectorTypeModel


class DuplicateProcessingInfoException(BusinessException):
    def __init__(
            self,
            user_id: UUID,
            connector_type: ConnectorTypeModel | None = None,
            processing_info_source: str | None = None
    ):
        message = f'Found more than one processing info for user with id = {user_id}'
        if connector_type:
            message += f" for connector type {connector_type}"

        if processing_info_source:
            message += f" in source {processing_info_source}"

        super().__init__(BusinessErrorCode.DuplicateUserProcessingInfo, message)
