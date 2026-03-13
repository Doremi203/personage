namespace Personage.Auth.Domain.Configuration;

public class JwtSettings
{
    public int AccessTokenTtlMinutes { get; init; }
    public int RefreshTokenTtlHours { get; init; }
    public string Issuer { get; init; } = null!;
    public string Audience { get; init; } = null!;

    /// <summary>
    /// RSA private key in PEM format for signing JWTs.
    /// In production, resolved from Lockbox via <c>secret:{id}:{version}:{key}</c>.
    /// </summary>
    public string PrivateKeyPem { get; set; } = null!;
}
