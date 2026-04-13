namespace Personage.Auth.Domain.Models.Common;

public class OAuthTokenModel
{
    public string AccessToken { get; init; } = null!;
    public string? RefreshToken { get; init; }
    public DateTime ExpiresAt { get; init; }
    public string GmailEmail { get; init; } = null!;
}