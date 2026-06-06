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
    /// Default data-plane endpoint of Yandex Cloud Lockbox (GetPayload). Talking to it
    /// directly lets us skip the SDK's endpoint-discovery call to api.cloud.yandex.net.
    /// Overridable via the <c>YandexCloud:LockboxPayloadEndpoint</c> configuration key.
    /// </summary>
    private const string DefaultPayloadEndpoint = "payload.lockbox.api.cloud.yandex.net:443";

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
        // Check if any values actually need resolution before acquiring credentials.
        // This avoids requiring Yandex Cloud credentials in development environments.
        var tempConfig = builder.Build();
        var hasSecrets = tempConfig.AsEnumerable()
            .Any(kv => kv.Value is not null &&
                        kv.Value.StartsWith(LockboxSecretConfigurationProvider.SecretPrefix, StringComparison.Ordinal));

        if (!hasSecrets)
            return builder;

        ICredentialsProvider credentialsProvider;
        if (useMetadataService)
        {
            credentialsProvider = new MetadataCredentialsProvider();
        }
        else
        {
            const string localSecretsPath = "../../../secrets.env";
            Env.Load(localSecretsPath);
            var iamToken = Environment.GetEnvironmentVariable("YC_TOKEN")
                           ?? throw new InvalidOperationException(
                               "YC_TOKEN environment variable is required when secret: references are present " +
                               "and YandexCloud:UseMetadataService is false");

            credentialsProvider = new Extensions.IamTokenCredentialsProvider(iamToken);
        }

        var payloadEndpoint = tempConfig["YandexCloud:LockboxPayloadEndpoint"] ?? DefaultPayloadEndpoint;

        builder.Add(new LockboxSecretConfigurationSource(credentialsProvider, payloadEndpoint));
        return builder;
    }
}
