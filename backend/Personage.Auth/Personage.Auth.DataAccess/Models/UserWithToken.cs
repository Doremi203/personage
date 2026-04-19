namespace Personage.Auth.DataAccess.Models;

public class UserWithToken
{
    public Guid UserId { get; init; }
    public string UserEmail { get; init; } = null!;
    public ShortOAuthToken Token { get; set; } = null!;
}

public class ShortOAuthToken
{
    public Guid TokenId { get; init; }
    public string AccessToken { get; set; } = null!;
    public string RefreshToken { get; set; } = null!;
    public DateTime ExpiresAt { get; set; }
    public string GmailEmail { get; init; } = null!;
    public DateTime? LastProcessedAt { get; init; }
}
