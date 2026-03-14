using Personage.Auth.Domain.Models.TelegramAuth;
using Personage.Auth.Domain.Models.TelegramAuth.Requests;

namespace Personage.Auth.Domain.Interfaces;

public interface ITelegramAuthService
{
    Task StoreSession(StoreSessionRequestModel request, CancellationToken ct);
    Task<TelegramSessionModel> GetSession(Guid userId, CancellationToken ct);
}