using Personage.Auth.Domain.Models.Common;

namespace Personage.Auth.Domain.Interfaces;

public interface IGoogleOAuthService
{
    string GetAuthorizationUrl(string redirectUri, string state);
    Task<GmailTokenModel> ExchangeCode(string code, string redirectUri, CancellationToken ct);
    Task<GmailTokenModel> RefreshToken(string refreshToken, CancellationToken ct);
}