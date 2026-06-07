using Personage.Auth.DataAccess.Models;

namespace Personage.Auth.DataAccess.Interfaces.Repositories;

public interface IGoogleCalendarTokenRepository : IOAuthRepositoryBase
{
    Task SaveToken(OAuthToken token, CancellationToken ct);
    Task MarkUsersAsProcessed((Guid UserId, DateTime ProcessedAt)[] users, CancellationToken ct);
    Task<OAuthTokenWithId?> GetTokenByUserId(Guid userId, CancellationToken ct);
    Task RemoveToken(Guid userId, CancellationToken ct);
}