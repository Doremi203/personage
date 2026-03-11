using Personage.Auth.DataAccess.Models;

namespace Personage.Auth.DataAccess.Interfaces.Repositories;

public interface IPasswordResetTokenRepository
{
    Task<PasswordResetToken?> GetToken(string token, CancellationToken ct);
    Task<PasswordResetToken> CreateToken(Guid userId, string token, DateTime expiresAt, CancellationToken ct);
    Task InvalidateToken(string token, CancellationToken ct);
    Task InvalidateAllUserTokens(Guid userId, CancellationToken ct);
}