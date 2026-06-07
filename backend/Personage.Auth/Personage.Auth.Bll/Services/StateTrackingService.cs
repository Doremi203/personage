using Microsoft.Extensions.Logging;
using Microsoft.Extensions.Options;
using Personage.Auth.DataAccess.Interfaces.Repositories;
using Personage.Auth.DataAccess.Models;
using Personage.Auth.Domain.Configuration;
using Personage.Auth.Domain.Exceptions;
using Personage.Auth.Domain.Exceptions.Base;
using Personage.Auth.Domain.Interfaces;
using Personage.Auth.Domain.Models.Common;
using Personage.Auth.Domain.Models.StateTracking;
using Personage.Auth.Domain.Models.StateTracking.Requests;
using Personage.Auth.Domain.Models.StateTracking.Responses;
using Personage.Auth.Domain.Models.TelegramAuth;

namespace Personage.Auth.Bll.Services;

public class StateTrackingService(
    IGmailTokenRepository gmailTokenRepository,
    ITelegramSessionRepository telegramSessionRepository,
    ITelegramChatRepository telegramChatRepository,
    IGoogleCalendarTokenRepository googleCalendarTokenRepository,
    IGoogleOAuthService googleOAuthService,
    IUserRepository userRepository,
    ILogger<StateTrackingService> logger,
    IOptions<ExternalClientOptions> externalClientOptions
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
            ServiceTypeModel.GoogleCalendar => await GetUsersForGoogleCalendarProcessing(request.BatchSize,
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

        var failedRefreshUsers = await RefreshExpiringTokensAndRemoveFailed(users, ServiceTypeModel.Gmail, ct);

        return new GetUsersForProcessingResponseModel
        {
            Users = users
                .Except(failedRefreshUsers)
                .Select(user => MapOAuthUser(user, ServiceTypeModel.Gmail))
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

        var chatsByUser = await telegramChatRepository.GetChatsForUsers(
            users.Select(u => u.UserId).ToArray(), ct);

        return new GetUsersForProcessingResponseModel
        {
            Users = users
                .Select(u => MapTelegramUser(u, chatsByUser))
                .ToArray()
        };
    }

    private async Task<GetUsersForProcessingResponseModel> GetUsersForGoogleCalendarProcessing(
        int batchSize,
        DateTime processedUntilMoment,
        CancellationToken ct
    )
    {
        var users = await userRepository.GetUsersGoogleCalendarProcessedBeforeMoment(
            processedUntilMoment, batchSize, ct);
        var failedRefreshUsers = await RefreshExpiringTokensAndRemoveFailed(users, ServiceTypeModel.GoogleCalendar, ct);
        
        return new GetUsersForProcessingResponseModel
        {
            Users = users
                .Except(failedRefreshUsers)
                .Select(user => MapOAuthUser(user, ServiceTypeModel.GoogleCalendar))
                .ToArray()
        };
    }

    private IOAuthRepositoryBase GetOAuthRepository(ServiceTypeModel oauthServiceType)
    {
        var invalidServiceException = new ArgumentOutOfRangeException(nameof(oauthServiceType), oauthServiceType,
            $@"'{oauthServiceType}' is not a valid OAuth service type.");
        IOAuthRepositoryBase oauthRepository = oauthServiceType switch
        {
            ServiceTypeModel.Unknown => throw invalidServiceException,
            ServiceTypeModel.Gmail => gmailTokenRepository,
            ServiceTypeModel.Telegram => throw invalidServiceException,
            ServiceTypeModel.GoogleCalendar => googleCalendarTokenRepository,
            _ => throw invalidServiceException
        };

        return oauthRepository;
    }
    
    private async Task<List<UserWithToken>> RefreshExpiringTokensAndRemoveFailed(
        UserWithToken[] users,
        ServiceTypeModel oauthServiceType,
        CancellationToken ct
    )
    {
        var expiringTokens = users
            .Where(x => x.Token.ExpiresAt <= DateTime.UtcNow.AddMinutes(TokenExpirationThresholdMinutes))
            .ToList();
        
        var refreshTasks = expiringTokens.Select(async expiringToken =>
        {
            try
            {
                await RefreshTokenAndUpdate(expiringToken, oauthServiceType, ct);
                return (Success: true, User: expiringToken);
            }
            catch (Exception ex)
            {
                logger.LogError(ex,
                    "Failed to refresh token of kind {ServiceKind} for user {UserEmail}",
                    oauthServiceType, expiringToken.UserEmail);
                return (Success: false, User: expiringToken);
            }
        });

        var results = await Task.WhenAll(refreshTasks);
        if (results.Length == 0)
            return [];
        var refreshFailed = results.Where(r => !r.Success).Select(r => r.User).ToList();

        var oauthRepository = GetOAuthRepository(oauthServiceType);
        var failedAttemptsInfo = await oauthRepository
            .UpdateRefreshInfo(results
                    .Select(x => (x.User.Token.TokenId, x.Success))
                    .ToArray(),
                ct
            );

        var maxAllowedRefreshAttempts = externalClientOptions.Value.MaxRefreshRetryAttempts;
        var tokensToRemove = failedAttemptsInfo
            .Where(token => token.FailedAttempts > maxAllowedRefreshAttempts)
            .Select(token => token.TokenId)
            .ToArray();

        if (tokensToRemove.Length > 0)
        {
            logger.LogError("Unable to refresh tokens of kind {ServiceKind}: {@TokensToRemove}. Removing tokens...", 
                oauthServiceType,
                tokensToRemove);
            await oauthRepository.RemoveTokens(tokensToRemove, ct);
        }

        return refreshFailed;
    }
    
    private async Task RefreshTokenAndUpdate(UserWithToken user, ServiceTypeModel oauthServiceType, CancellationToken ct)
    {
        var refreshedToken = await googleOAuthService.RefreshToken(
            user.Token.RefreshToken, ct);

        var newRefreshToken = refreshedToken.RefreshToken ?? user.Token.RefreshToken;

        var oauthRepository = GetOAuthRepository(oauthServiceType);
        await oauthRepository.UpdateToken(
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

            case ServiceTypeModel.GoogleCalendar:
                await googleCalendarTokenRepository.MarkUsersAsProcessed(usersToMark, ct);
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

    private static UserForProcessingModel MapOAuthUser(UserWithToken model, ServiceTypeModel serviceType)
    {
        var tokens = new OAuthTokenModel
        {
            AccessToken = model.Token.AccessToken,
            RefreshToken = model.Token.RefreshToken,
            ExpiresAt = model.Token.ExpiresAt,
            GmailEmail = model.Token.GmailEmail,
        };
        return serviceType switch
        {
            ServiceTypeModel.GoogleCalendar => new UserForProcessingModel
            {
                UserId = model.UserId,
                LastProcessedAt = model.Token.LastProcessedAt,
                Credentials = new GoogleCalendarProcessingCredentials { Tokens = tokens }
            },
            ServiceTypeModel.Gmail => new UserForProcessingModel
            {
                UserId = model.UserId,
                LastProcessedAt = model.Token.LastProcessedAt,
                Credentials = new GmailProcessingCredentials { Tokens = tokens }
            },
            _ => throw new ArgumentOutOfRangeException(nameof(serviceType), serviceType, null)
        };
    }

    private static UserForProcessingModel MapTelegramUser(
        UserWithTelegramSession model,
        IReadOnlyDictionary<Guid, IReadOnlyList<TelegramChatModel>> chatsByUser)
    {
        var chats = chatsByUser.TryGetValue(model.UserId, out var userChats)
            ? userChats
            : [];

        return new UserForProcessingModel
        {
            UserId = model.UserId,
            LastProcessedAt = model.LastProcessedAt,
            Credentials = new TelegramProcessingCredentials
            {
                SessionString = model.SessionString,
                Chats = chats
            }
        };
    }
}