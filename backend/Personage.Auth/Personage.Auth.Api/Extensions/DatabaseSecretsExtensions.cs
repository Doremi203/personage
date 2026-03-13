using Microsoft.Extensions.Options;
using Npgsql;
using Personage.Auth.Domain.Configuration;
using Yandex.Cloud;
using Yandex.Cloud.Lockbox.V1;

namespace Personage.Auth.Api.Extensions;

public static class DatabaseSecretsExtensions
{
    /// <summary>
    /// Resolves the database password from Yandex Lockbox and patches the connection string
    /// in <see cref="ConnectionFactorySettings"/>. This is called once at application startup.
    /// If <see cref="ConnectionFactorySettings.SecretId"/> is not configured, this method is a no-op
    /// (e.g., in development where the password is in the connection string directly).
    /// </summary>
    public static async Task ResolveDatabaseSecrets(this WebApplication app)
    {
        var settings = app.Services.GetRequiredService<IOptions<ConnectionFactorySettings>>().Value;

        if (string.IsNullOrEmpty(settings.SecretId))
            return;

        var logger = app.Services.GetRequiredService<ILogger<Program>>();
        var sdk = app.Services.GetRequiredService<Sdk>();

        try
        {
            logger.LogInformation("Resolving database password from Lockbox secret {SecretId}", settings.SecretId);

            var payload = await sdk.Services.Lockbox.PayloadService.GetAsync(
                new GetPayloadRequest { SecretId = settings.SecretId }
            );

            var password = payload.Entries
                .SingleOrDefault(e => e.Key == settings.PasswordKey)?.TextValue
                ?? throw new InvalidOperationException(
                    $"Key '{settings.PasswordKey}' not found in Lockbox secret '{settings.SecretId}'");

            var builder = new NpgsqlConnectionStringBuilder(settings.ConnectionString)
            {
                Password = password
            };
            settings.ConnectionString = builder.ConnectionString;

            logger.LogInformation("Database password resolved successfully from Lockbox");
        }
        catch (Exception ex)
        {
            logger.LogError(ex, "Failed to resolve database password from Lockbox secret {SecretId}", settings.SecretId);
            throw;
        }
    }
}
