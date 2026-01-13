namespace Personage.Auth.DataAccess.Models;

public class GmailToken
{
    public Guid UserId { get; init; }
    public string AccessToken { get; init; } = null!;
    public string RefreshToken { get; init; } = null!;
    public DateTime ExpiresAt { get; init; }
    public string GmailEmail { get; init; } = null!;
    public DateTime CreatedAt { get; init; }
}