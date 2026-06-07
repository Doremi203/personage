using Personage.Auth.DataAccess.Models;

namespace Personage.Auth.DataAccess.Interfaces.Repositories;

public interface IGmailTokenRepository : IOAuthRepositoryBase
{
    Task<OAuthTokenWithId?> GetTokenByUserEmail(string userEmail, CancellationToken ct);
    Task<OAuthTokenWithId?> GetTokenByUserId(Guid userId, CancellationToken ct);
    Task SaveToken(OAuthToken token, CancellationToken ct);
    Task<Guid[]> GetUsersWithoutToken(Guid[] userIds, CancellationToken ct);
    Task MarkUsersAsProcessed((Guid UserId, DateTime ProcessedAt)[] users, CancellationToken ct);
    Task RemoveToken(Guid userId, CancellationToken ct);
}