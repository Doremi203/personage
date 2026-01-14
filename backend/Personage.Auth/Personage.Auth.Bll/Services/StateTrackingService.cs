using Personage.Auth.DataAccess.Interfaces.Repositories;
using Personage.Auth.DataAccess.Models;
using Personage.Auth.Domain.Exceptions;
using Personage.Auth.Domain.Interfaces;
using Personage.Auth.Domain.Models.Common;
using Personage.Auth.Domain.Models.StateTracking;
using Personage.Auth.Domain.Models.StateTracking.Requests;
using Personage.Auth.Domain.Models.StateTracking.Responses;

namespace Personage.Auth.Bll.Services;

public class StateTrackingService(IUserRepository userRepository) : IStateTrackingService
{
    public async Task<GetUsersForProcessingResponseModel> GetUsersForProcessing(GetUsersForProcessingRequestModel request, CancellationToken ct)
    {
        if(request.ServiceType is not ServiceTypeModel.Gmail)
            throw new ServiceTypeNotSupportedException($"Service type {request.ServiceType} is not supported for {nameof(GetUsersForProcessing)}");
        
        var processedUntilMoment = DateTime.UtcNow.AddSeconds(-request.MinSecondsSinceLastProcess); 
        var users = await userRepository.GetUsersProcessedBeforeMoment(processedUntilMoment, request.BatchSize, ct);

        return new GetUsersForProcessingResponseModel
        {
            Users = users.Select(Map).ToArray() 
        };
    }

    public Task MarkUsersAsProcessed(MarkUsersAsProcessedRequestModel request, CancellationToken ct)
    {
        throw new NotImplementedException();
    }

    private static UserForProcessingModel Map(UserWithToken model)
    {
        return new UserForProcessingModel
        {
            UserId = model.UserId,
            UserEmail = model.UserEmail,
            Tokens = new GmailTokenModel
                {
                    AccessToken = model.Token.AccessToken,
                    RefreshToken = model.Token.RefreshToken,
                    ExpiresAt = model.Token.ExpiresAt,
                    GmailEmail = model.Token.GmailEmail,
                },
            LastProcessedAt = model.Token.LastProcessedAt
        };
    }
}