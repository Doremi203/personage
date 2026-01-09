namespace Personage.Auth.Domain.Models.Responses;

public class TokenExchangeResultModel
{
    public string AccessToken { get; init; } = null!;
    public string RefreshToken { get; init; } = null!;
    public DateTime ExpiresAt { get; init; }
    public string GmailEmail { get; init; } = null!;
}