namespace Personage.Auth.Domain.Models.Common;

public class GmailTokenModel
{
    public string AccessToken { get; init; } = null!;
    public string RefreshToken { get; init; } = null!;
    public DateTime ExpiresAt { get; init; }
    public string GmailEmail { get; init; } = null!;
}