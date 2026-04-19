using Personage.Auth.DataAccess.Models.Requests;

namespace Personage.Auth.DataAccess.Models;

public record OAuthToken
{
    public Guid UserId { get; init; }
    public string AccessToken { get; init; } = null!;
    public string RefreshToken { get; init; } = null!;
    public DateTime ExpiresAt { get; init; }
    public string GmailEmail { get; init; } = null!;
    public OAuthTokenStatus Status { get; init; }
}

public record OAuthTokenWithId : OAuthToken
{
    public Guid Id { get; init; }
}