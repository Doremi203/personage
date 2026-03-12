namespace Personage.Auth.Domain.Configuration;

public class ConnectionFactorySettings
{
    public string ConnectionString { get; set; } = null!;
    public string? SecretId { get; init; }
    public string PasswordKey { get; init; } = "postgresql_password";
}