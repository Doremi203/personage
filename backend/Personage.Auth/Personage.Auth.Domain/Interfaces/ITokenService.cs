using Personage.Auth.Domain.Models.Common;

namespace Personage.Auth.Domain.Interfaces;

public interface ITokenService
{
    Task<PersonageTokenModel> RefreshAccessToken(string refreshToken, CancellationToken ct);
    Task<GmailTokenModel> GetUserGmailToken(string userEmail, CancellationToken ct);
    bool VerifyToken(string token);
    Task<RefreshTokenModel> GenerateAndStoreRefreshToken(Guid userId, CancellationToken ct);
    string GenerateAccessToken(Guid userId);
}