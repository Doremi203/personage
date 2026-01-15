using Microsoft.Extensions.Logging;
using Personage.Auth.DataAccess.Interfaces.Repositories;
using Personage.Auth.DataAccess.Models;
using Personage.Auth.Domain.Exceptions;
using Personage.Auth.Domain.Interfaces;
using Personage.Auth.Domain.Models.Common;
using Personage.Auth.Domain.Models.StateTracking;
using Personage.Auth.Domain.Models.StateTracking.Requests;
using Personage.Auth.Domain.Models.StateTracking.Responses;

namespace Personage.Auth.Bll.Services;

public class StateTrackingService(
    IGmailTokenRepository gmailTokenRepository,
    IUserRepository userRepository,
    ILogger<StateTrackingService> logger
) : IStateTrackingService
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

    public async Task MarkUsersAsProcessed(MarkUsersAsProcessedRequestModel request, CancellationToken ct)
    {
        if(request.ServiceType is not ServiceTypeModel.Gmail)
            throw new ServiceTypeNotSupportedException($"Service type {request.ServiceType} is not supported for {nameof(MarkUsersAsProcessed)}");
        
        var userIds = request.Users.Select(u => u.UserId).ToArray();
        var duplicatedUsers = userIds
            .GroupBy(x => x)
            .Where(g => g.Count() > 1)
            .Select(g => g.Key)
            .ToArray();

        if (duplicatedUsers.Length != 0)
            throw new CustomException(
                ErrorCode.DuplicatedUsersForbidden,
                $"Users must be unique. Duplicated users: [{string.Join(", ", duplicatedUsers)}]");

        var usersWithoutToken = await gmailTokenRepository.GetUsersWithoutToken(userIds, ct);
        if(usersWithoutToken.Length != 0)
        {
            logger.LogError("Attempt to mark users processing for users with no gmail access: {@UsersWithoutToken}", usersWithoutToken);
            throw new CustomException(
                ErrorCode.UsersNotAuthorizedForProcessing,
                "Cannot mark users gmail processing, the following users have no gmail access:" +
                "\n" + string.Join("\n", usersWithoutToken)
            );
        }

        await userRepository.MarkUsersAsProcessed(
            request.Users.Select(x => (x.UserId, x.ProcessedAt)).ToArray(), ct);
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