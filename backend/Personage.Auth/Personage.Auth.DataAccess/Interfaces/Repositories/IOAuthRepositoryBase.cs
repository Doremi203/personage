namespace Personage.Auth.DataAccess.Interfaces.Repositories;

public interface IOAuthRepositoryBase
{
    Task RemoveTokens(Guid[] tokenIds, CancellationToken ct);
    Task UpdateToken(Guid tokenId, string accessToken, string refreshToken, DateTime expiresAt, CancellationToken ct);
    Task<(Guid TokenId, int FailedAttempts)[]> UpdateRefreshInfo((Guid TokenId, bool RefreshSuccess)[] refreshes,
        CancellationToken ct);
}