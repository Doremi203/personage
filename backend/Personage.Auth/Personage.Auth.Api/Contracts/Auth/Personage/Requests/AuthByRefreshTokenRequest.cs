namespace Personage.Auth.Api.Contracts.Auth.Personage.Requests;

public class AuthByRefreshTokenRequest
{
    public string RefreshToken { get; init; } = null!;
}