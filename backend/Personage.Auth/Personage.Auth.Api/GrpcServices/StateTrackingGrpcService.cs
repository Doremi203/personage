using Google.Protobuf.WellKnownTypes;
using Grpc.Core;
using Personage.Auth.Api.Grpc.Common;
using Personage.Auth.Api.Grpc.State;
using Personage.Auth.Api.Mappers;
using Personage.Auth.Domain.Interfaces;
using Personage.Auth.Domain.Models.Common;
using Personage.Auth.Domain.Models.StateTracking;
using Personage.Auth.Domain.Models.StateTracking.Requests;

namespace Personage.Auth.Api.GrpcServices;

public class StateTrackingGrpcService(
    IStateTrackingService stateTrackingService
) : StateTrackingService.StateTrackingServiceBase
{
    public override async Task<GetUsersForProcessingResponse> GetUsersForProcessing(GetUsersForProcessingRequest request, ServerCallContext context)
    {
        var res = await stateTrackingService.GetUsersForProcessing(
            new GetUsersForProcessingRequestModel
            {
                BatchSize = request.BatchSize,
                MinSecondsSinceLastProcess = request.MinSecondsSinceLastProcess,
                ServiceType = request.ServiceType switch
                {
                    ServiceType.Unknown => ServiceTypeModel.Unknown,
                    ServiceType.Gmail => ServiceTypeModel.Gmail,
                    _ => ServiceTypeModel.Unknown
                }
            },
            context.CancellationToken);

        return new GetUsersForProcessingResponse
        {
            Users = { res.Users.Select(Map) }
        };
    }

    public override async Task<MarkUsersAsProcessedResponse> MarkUsersAsProcessed(MarkUsersAsProcessedRequest request,
        ServerCallContext context)
    {
        await stateTrackingService.MarkUsersAsProcessed(
            new MarkUsersAsProcessedRequestModel
            {
                Users = request.Users.Select(Map).ToArray()
            }, context.CancellationToken);
        return new MarkUsersAsProcessedResponse();
    }

    private static ProcessedUserModel Map(ProcessedUser grpcProcessedUser)
    {
        return new ProcessedUserModel
        {
            UserId = Guid.Parse(grpcProcessedUser.UserId),
            ProcessedAt = grpcProcessedUser.ProcessedAt.ToDateTime()
        };
    }
    
    private static UserForProcessing Map(UserForProcessingModel model)
    {
        var res = new UserForProcessing
        {
            UserId = model.UserId.ToString(),
            UserEmail = model.UserEmail,
            GmailEmail = model.GmailEmail,
            Tokens = CommonMapper.ToGrpcGmailTokens(model.Tokens)
        };
        
        if(model.LastProcessedAt is not null)
            res.LastProcessedAt = Timestamp.FromDateTime(model.LastProcessedAt.Value);

        return res;
    }
}