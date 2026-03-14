using DotNetEnv;
using Yandex.Cloud;
using Yandex.Cloud.Credentials;

namespace Personage.Auth.Api.Configuration;

/// <summary>
/// Extension methods for adding Lockbox secret resolution to the configuration pipeline.
/// </summary>
public static class LockboxConfigurationExtensions
{
    /// <summary>
    /// Adds a configuration source that resolves <c>secret:{id}:{version}:{key}</c>
    /// references from Yandex Cloud Lockbox.
    ///
    /// Must be called after all other configuration sources have been added.
    /// In non-production environments where no <c>secret:</c> values exist,
    /// this is effectively a no-op (Lockbox is never contacted).
    /// </summary>
    public static IConfigurationBuilder AddLockboxSecrets(
        this IConfigurationBuilder builder,
        bool useMetadataService)
    {
        // Check if any values actually need resolution before creating the SDK client.
        // This avoids requiring Yandex Cloud credentials in development environments.
        var tempConfig = builder.Build();
        var hasSecrets = tempConfig.AsEnumerable()
            .Any(kv => kv.Value is not null &&
                        kv.Value.StartsWith(LockboxSecretConfigurationProvider.SecretPrefix, StringComparison.Ordinal));

        if (!hasSecrets)
            return builder;

        Sdk sdk;
        if (useMetadataService)
        {
            sdk = new Sdk(new MetadataCredentialsProvider());
        }
        else
        {
            const string localSecretsPath = "../../../secrets.env";
            Env.Load(localSecretsPath);
            var iamToken = Environment.GetEnvironmentVariable("YC_TOKEN")
                           ?? throw new InvalidOperationException(
                               "YC_TOKEN environment variable is required when secret: references are present " +
                               "and YandexCloud:UseMetadataService is false");

            sdk = new Sdk(new Extensions.IamTokenCredentialsProvider(iamToken));
        }

        builder.Add(new LockboxSecretConfigurationSource(sdk));
        return builder;
    }
}
