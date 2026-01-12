using Google.Protobuf.WellKnownTypes;
using Grpc.Core;
using Personage.Auth.Api.Grpc;

namespace Personage.Auth.Api.GrpcServices;

public class TestGrpcService : TestService.TestServiceBase
{
    public override Task<PingResponse> Ping(PingRequest request, ServerCallContext context)
    {
        return Task.FromResult(new PingResponse
        {
            Message = "Pong",
            Moment = Timestamp.FromDateTime(DateTime.UtcNow)
        });
    }
}