from datetime import datetime, timezone, timedelta
from uuid import UUID

from app.domain.interfaces.business_logic.ICommonProcessingService import ICommonProcessingService
from proto import processing_pb2_grpc, processing_pb2
from grpc import ServicerContext

class ProcessingGrpcService(processing_pb2_grpc.ProcessingServiceServicer):
    def __init__(
            self,
            processing_service: ICommonProcessingService
    ):
        self.processing_service = processing_service

    async def SendProcessingSnapshot(
            self,
            request: processing_pb2. SendProcessingSnapshotRequest,
            context: ServicerContext
    ) -> processing_pb2.SendProcessingSnapshotResponse:
        target_queue_url = request.target_queue_url if request.HasField("target_queue_url") else None
        sent_events_count = await self.processing_service.resend_processing_snapshot(
            snapshot_id=UUID(request.snapshot_id),
            target_queue_url=target_queue_url,
        )
        return processing_pb2.SendProcessingSnapshotResponse(events_count=sent_events_count)

    async def CreateProcessingSnapshot(
            self,
            request: processing_pb2.CreateProcessingSnapshotRequest,
            context: ServicerContext
    ) -> processing_pb2.CreateProcessingSnapshotResponse:
        snapshot_id = await self.processing_service.create_processing_snapshot(
            from_=request.from_,
            to=request.to,
        )
        return processing_pb2.CreateProcessingSnapshotResponse(snapshot_id=str(snapshot_id))

    async def CreateSnapshotFromCurrentMoment(
            self,
            request: processing_pb2.CreateSnapshotFromCurrentMomentRequest,
            context: ServicerContext
    ) -> processing_pb2.CreateSnapshotFromCurrentMomentResponse:
        start = datetime.now(timezone.utc)
        end = start + timedelta(seconds=request.for_seconds)
        snapshot_id = await self.processing_service.create_processing_snapshot(
            from_=start,
            to=end,
        )
        return processing_pb2.CreateProcessingSnapshotResponse(snapshot_id=str(snapshot_id))

    async def GetProcessingSnapshots(
            self,
            request: processing_pb2.GetProcessingSnapshotsRequest,
            context: ServicerContext
    ) -> processing_pb2.GetProcessingSnapshotsResponse:
        snapshots = await self.processing_service.get_processing_snapshots()
        return processing_pb2.GetProcessingSnapshotsResponse(
            snapshots=[processing_pb2.ProcessingSnapshot(
                id=str(snapshot.id),
                from_=snapshot.from_,
                to=snapshot.to
            ) for snapshot in snapshots]
        )
