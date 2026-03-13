from datetime import datetime
from uuid import UUID

from google.protobuf.timestamp_pb2 import Timestamp

from app.domain.models.users.GmailTokensModel import GmailTokensModel
from app.domain.models.users.ProcessedUserModel import ProcessedUserModel
from app.domain.models.users.UserForGmailProcessingModel import UserForGmailProcessingModel
from externalClients.BaseGrpcClient import BaseGrpcClient
from externalClients.personage_auth.proto import state_tracking_pb2_grpc
from externalClients.personage_auth.proto.common_pb2 import (GmailTokens)
from externalClients.personage_auth.proto.common_pb2 import (ServiceType)
from externalClients.personage_auth.proto.state_tracking_pb2 import (
    GetUsersForProcessingRequest,
    GetUsersForProcessingResponse,
    UserForProcessing,
    MarkUsersAsProcessedRequest,
    ProcessedUser
)


class StateTrackingClient(BaseGrpcClient):
    def __init__(
            self,
            endpoint: str,
            use_tls: bool = False,
    ):
        super().__init__(endpoint, use_tls=use_tls)
        self._stub = state_tracking_pb2_grpc.StateTrackingServiceStub(self._channel)

    async def get_users_for_processing(
            self,
            batch_size: int,
            seconds_since_last_process: int
    ) -> list[UserForGmailProcessingModel]:
        request = GetUsersForProcessingRequest(
            batch_size=batch_size,
            min_seconds_since_last_process=seconds_since_last_process,
            service_type=ServiceType.ServiceType_Gmail
        )

        response: GetUsersForProcessingResponse = await self._stub.GetUsersForProcessing(request)
        return [StateTrackingClient.__to_domain_user(user) for user in response.users]

    async def mark_processed_users(
            self,
            processed_users: list[ProcessedUserModel],
    ) -> None:
        request = MarkUsersAsProcessedRequest(
            service_type=ServiceType.ServiceType_Gmail,
            users=[StateTrackingClient.__to_grpc_processed_user(user) for user in processed_users]
        )
        await self._stub.MarkUsersAsProcessed(request)

    @staticmethod
    def __to_domain_user(user: UserForProcessing) -> UserForGmailProcessingModel:
        return UserForGmailProcessingModel(
            user_id=UUID(user.user_id),
            user_email=user.user_email,
            tokens=StateTrackingClient.__to_domain_tokens(user.tokens) if user.HasField('tokens') else None,
            last_processed_at=user.last_processed_at.ToDatetime() if user.HasField('last_processed_at') else None,
        )

    @staticmethod
    def __to_domain_tokens(tokens: GmailTokens) -> GmailTokensModel:
        return GmailTokensModel(
            access_token=tokens.access_token,
            refresh_token=tokens.refresh_token,
            expires_at=tokens.expires_at.ToDatetime() if tokens.HasField('expires_at') else None,
            gmail_email=tokens.gmail_email,
        )

    @staticmethod
    def __to_grpc_processed_user(user: ProcessedUserModel) -> ProcessedUser:
        return ProcessedUser(
            user_id=str(user.user_id),
            processed_at=StateTrackingClient.__to_grpc_timestamp(user.processed_at)
        )

    @staticmethod
    def __to_grpc_timestamp(date: datetime) -> Timestamp:
        timestamp = Timestamp()
        timestamp.FromDatetime(date)
        return timestamp
