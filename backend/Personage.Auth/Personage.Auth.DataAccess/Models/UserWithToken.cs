namespace Personage.Auth.DataAccess.Models;

public class UserWithToken
{
    public Guid UserId { get; init; }
    public string UserEmail { get; init; } = null!;
    public ShortGmailToken Token { get; set; } = null!;
}

public class ShortGmailToken
{
    public string AccessToken { get; init; } = null!;
    public string RefreshToken { get; init; } = null!;
    public DateTime ExpiresAt { get; init; }
    public string GmailEmail { get; init; } = null!;
    public DateTime? LastProcessedAt { get; init; }
}
