using Personage.Auth.Domain.Models.Responses;

namespace Personage.Auth.Domain.Interfaces;

public interface IGoogleOAuthService
{
    string GetAuthorizationUrl(string redirectUri, string state);
    Task<TokenExchangeResultModel> ExchangeCode(string code, string redirectUri, CancellationToken ct);
}