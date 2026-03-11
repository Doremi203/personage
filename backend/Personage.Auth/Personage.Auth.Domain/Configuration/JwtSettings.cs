namespace Personage.Auth.Domain.Configuration;

public class JwtSettings
{
    public int AccessTokenTtlMinutes { get; init; }
    public int RefreshTokenTtlHours { get; init; }
    public string Issuer { get; init; } = null!;
    public string Audience { get; init; } = null!;
    public string? PrivateKey { get; init; }
}