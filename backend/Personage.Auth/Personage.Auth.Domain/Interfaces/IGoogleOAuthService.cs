using Personage.Auth.Domain.Models.Common;
using Personage.Auth.Domain.Models.GoogleAuth;

namespace Personage.Auth.Domain.Interfaces;

public interface IGoogleOAuthService
{
    string GetAuthorizationUrl(string redirectUri, string state, GoogleServiceKind serviceKind);
    Task<OAuthTokenModel> ExchangeCode(string code, string redirectUri, CancellationToken ct);
    Task<OAuthTokenModel> RefreshToken(string refreshToken, CancellationToken ct);
}