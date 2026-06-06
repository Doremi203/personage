using Yandex.Cloud.Credentials;

namespace Personage.Auth.Api.Configuration;

/// <summary>
/// Configuration source that creates a <see cref="LockboxSecretConfigurationProvider"/>
/// to resolve <c>secret:{id}:{version}:{key}</c> references from Yandex Cloud Lockbox.
///
/// This source must be added last so it can post-process values from all preceding sources.
/// </summary>
public sealed class LockboxSecretConfigurationSource : IConfigurationSource
{
    private readonly ICredentialsProvider _credentialsProvider;
    private readonly string _payloadEndpoint;

    public LockboxSecretConfigurationSource(ICredentialsProvider credentialsProvider, string payloadEndpoint)
    {
        _credentialsProvider = credentialsProvider;
        _payloadEndpoint = payloadEndpoint;
    }

    public IConfigurationProvider Build(IConfigurationBuilder builder)
    {
        // Build the inner configuration from all sources added before this one.
        // This gives us access to all config values so we can scan for secret: prefixes.
        var innerConfig = new ConfigurationBuilder()
            .AddInMemoryCollection(
                builder.Build().AsEnumerable()
                    .Where(kv => kv.Value is not null)
                    .Select(kv => new KeyValuePair<string, string?>(kv.Key, kv.Value))
            )
            .Build();

        return new LockboxSecretConfigurationProvider(innerConfig, _credentialsProvider, _payloadEndpoint);
    }
}
