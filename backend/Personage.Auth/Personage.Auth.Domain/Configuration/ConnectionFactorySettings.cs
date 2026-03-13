namespace Personage.Auth.Domain.Configuration;

public class ConnectionFactorySettings
{
    public string ConnectionString { get; set; } = null!;

    /// <summary>
    /// Database password. In production, this is resolved from Lockbox via the
    /// <c>secret:{id}:{version}:{key}</c> format before the application starts.
    /// At startup, it is merged into <see cref="ConnectionString"/>.
    /// </summary>
    public string? Password { get; set; }
}
