using Microsoft.Extensions.Logging;
using Personage.Auth.DataAccess.Interfaces.Repositories;
using Personage.Auth.DataAccess.Models;
using Personage.Auth.Domain.Exceptions;
using Personage.Auth.Domain.Exceptions.Base;
using Personage.Auth.Domain.Interfaces;
using Personage.Auth.Domain.Models.Common;
using Personage.Auth.Domain.Models.StateTracking;
using Personage.Auth.Domain.Models.StateTracking.Requests;
using Personage.Auth.Domain.Models.StateTracking.Responses;

namespace Personage.Auth.Bll.Services;

public class StateTrackingService(
    IGmailTokenRepository gmailTokenRepository,
    ITelegramSessionRepository telegramSessionRepository,
    IGoogleOAuthService googleOAuthService,
    IUserRepository userRepository,
    ILogger<StateTrackingService> logger
) : IStateTrackingService
{
    private const int TokenExpirationThresholdMinutes = 5;

    public async Task<GetUsersForProcessingResponseModel> GetUsersForProcessing(
        GetUsersForProcessingRequestModel request, CancellationToken ct)
    {
        var processedUntilMoment = DateTime.UtcNow.AddSeconds(-request.MinSecondsSinceLastProcess);

        return request.ServiceType switch
        {
            ServiceTypeModel.Gmail => await GetUsersForGmailProcessing(request.BatchSize,
                processedUntilMoment, ct),
            ServiceTypeModel.Telegram => await GetUsersForTelegramProcessing(request.BatchSize,
                processedUntilMoment, ct),
            _ =>
                throw new ServiceTypeNotSupportedException(
                    $"Service type {request.ServiceType} is not supported for {nameof(GetUsersForProcessing)}")
        };
    }

    private async Task<GetUsersForProcessingResponseModel> GetUsersForGmailProcessing(
        int batchSize,
        DateTime processedUntilMoment,
        CancellationToken ct
    )
    {
        var users = await userRepository.GetUsersGmailProcessedBeforeMoment(
            processedUntilMoment, batchSize, ct);

        var expiringTokens = users
            .Where(x => x.Token.ExpiresAt <= DateTime.UtcNow.AddMinutes(TokenExpirationThresholdMinutes))
            .ToList();

        var refreshTasks = expiringTokens.Select(async expiringToken =>
        {
            try
            {
                await RefreshTokenAndUpdate(expiringToken, ct);
                return (Success: true, User: expiringToken);
            }
            catch (Exception ex)
            {
                logger.LogError(ex,
                    "Failed to refresh Gmail token for user {UserEmail}",
                    expiringToken.UserEmail);
                return (Success: false, User: expiringToken);
            }
        });

        var results = await Task.WhenAll(refreshTasks);
        var refreshFailed = results.Where(r => !r.Success).Select(r => r.User).ToList();

        return new GetUsersForProcessingResponseModel
        {
            Users = users
                .Except(refreshFailed)
                .Select(MapGmailUser)
                .ToArray()
        };
    }

    private async Task<GetUsersForProcessingResponseModel> GetUsersForTelegramProcessing(
        int batchSize,
        DateTime processedUntilMoment,
        CancellationToken ct
    )
    {
        var users = await userRepository.GetUsersTelegramProcessedBeforeMoment(
            processedUntilMoment, batchSize, ct);
        return new GetUsersForProcessingResponseModel
        {
            Users = users
                .Select(MapTelegramUser)
                .ToArray()
        };
    }

    private async Task RefreshTokenAndUpdate(UserWithToken user, CancellationToken ct)
    {
        var refreshedToken = await googleOAuthService.RefreshToken(
            user.Token.RefreshToken, ct);

        var newRefreshToken = refreshedToken.RefreshToken ?? user.Token.RefreshToken;

        await gmailTokenRepository.UpdateToken(
            user.Token.TokenId,
            refreshedToken.AccessToken,
            newRefreshToken,
            refreshedToken.ExpiresAt,
            ct);

        user.Token.AccessToken = refreshedToken.AccessToken;
        user.Token.RefreshToken = newRefreshToken;
        user.Token.ExpiresAt = refreshedToken.ExpiresAt;
    }

    public async Task MarkUsersAsProcessed(MarkUsersAsProcessedRequestModel request, CancellationToken ct)
    {
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

        var usersToMark = request.Users.Select(x => (x.UserId, x.ProcessedAt)).ToArray();
        switch (request.ServiceType)
        {
            case ServiceTypeModel.Gmail:
                await MarkGmailUsersAsProcessedAndVerify(usersToMark, ct);
                break;

            case ServiceTypeModel.Telegram:
                await telegramSessionRepository.MarkUsersAsProcessed(usersToMark, ct);
                break;

            case ServiceTypeModel.Unknown:
            default:
                throw new ServiceTypeNotSupportedException(
                    $"Service type {request.ServiceType} is not supported for {nameof(MarkUsersAsProcessed)}");
        }
    }

    private async Task MarkGmailUsersAsProcessedAndVerify((Guid UserId, DateTime ProcessedAt)[] users,
        CancellationToken ct)
    {
        var usersWithoutToken =
            await gmailTokenRepository.GetUsersWithoutToken(users.Select(x => x.UserId).ToArray(), ct);
        if (usersWithoutToken.Length != 0)
        {
            logger.LogError("Attempt to mark users processing for users with no gmail access: {@UsersWithoutToken}",
                usersWithoutToken);
            throw new CustomException(
                ErrorCode.UsersNotAuthorizedForProcessing,
                "Cannot mark users gmail processing, the following users have no gmail access:" +
                "\n" + string.Join("\n", usersWithoutToken)
            );
        }

        await gmailTokenRepository.MarkUsersAsProcessed(users, ct);
    }

    private static UserForProcessingModel MapGmailUser(UserWithToken model)
    {
        return new UserForProcessingModel
        {
            UserId = model.UserId,
            LastProcessedAt = model.Token.LastProcessedAt,
            Credentials = new GmailProcessingCredentials
            {
                Tokens = new OAuthTokenModel
                {
                    AccessToken = model.Token.AccessToken,
                    RefreshToken = model.Token.RefreshToken,
                    ExpiresAt = model.Token.ExpiresAt,
                    GmailEmail = model.Token.GmailEmail,
                },
            }
        };
    }

    private static UserForProcessingModel MapTelegramUser(UserWithTelegramSession model)
    {
        return new UserForProcessingModel
        {
            UserId = model.UserId,
            LastProcessedAt = model.LastProcessedAt,
            Credentials = new TelegramProcessingCredentials
            {
                SessionString = model.SessionString
            }
        };
    }
}