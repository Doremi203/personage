namespace Personage.Auth.DataAccess.Models;

public class OAuthState
{
    public string State { get; init; } = null!;
    public string UserEmail { get; init; } = null!;
    public string RedirectUri { get; init; } = null!;
    public DateTime CreatedAt { get; init; }
    public DateTime ExpiresAt { get; init; }
}