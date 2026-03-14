using Personage.Auth.DataAccess.Models;

namespace Personage.Auth.DataAccess.Interfaces.Repositories;

public interface IGmailTokenRepository
{
    Task<GmailTokenWithId?> GetTokenByUserEmail(string userEmail, CancellationToken ct);
    Task SaveToken(GmailToken token, CancellationToken ct);
    Task UpdateToken(Guid tokenId, string accessToken, string refreshToken, DateTime expiresAt, CancellationToken ct);
    Task<Guid[]> GetUsersWithoutToken(Guid[] userIds, CancellationToken ct);
    Task MarkUsersAsProcessed((Guid UserId, DateTime ProcessedAt)[] users, CancellationToken ct);
}