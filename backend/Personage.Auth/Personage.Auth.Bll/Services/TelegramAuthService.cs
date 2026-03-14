using Personage.Auth.DataAccess.Interfaces.Repositories;
using Personage.Auth.Domain.Exceptions.Base;
using Personage.Auth.Domain.Interfaces;
using Personage.Auth.Domain.Models.TelegramAuth;
using Personage.Auth.Domain.Models.TelegramAuth.Requests;

namespace Personage.Auth.Bll.Services;

public class TelegramAuthService(
    ITelegramSessionRepository repository
) : ITelegramAuthService
{
    public async Task StoreSession(StoreSessionRequestModel request, CancellationToken ct)
    {
        await repository.StoreSession(request.UserId, request.SessionString, ct);
    }

    public async Task<TelegramSessionModel> GetSession(Guid userId, CancellationToken ct)
    {
        if (await repository.GetSessionString(userId, ct) is not { } sessionString)
            throw new NotFoundException(ErrorCode.TelegramSessionNotFound,
                $"Telegram session not found for user {userId}");

        return new TelegramSessionModel
        {
            SessionString = sessionString
        };
    }
}