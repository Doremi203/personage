using Yandex.Cloud;
using Yandex.Cloud.Lockbox.V1;

namespace Personage.Auth.Api.Configuration;

/// <summary>
/// A configuration provider that resolves values prefixed with <c>secret:</c>
/// from Yandex Cloud Lockbox.
///
/// Format: <c>secret:{secret_id}:{version_id}:{key}</c>
///
/// This mirrors the convention used by Go services (webapp/config.go) and
/// the traitex Python service (config.py), providing a unified approach
/// to secret management across the entire Personage platform.
/// </summary>
public sealed class LockboxSecretConfigurationProvider : ConfigurationProvider
{
    public const string SecretPrefix = "secret:";

    private readonly IConfigurationRoot _innerConfig;
    private readonly Sdk _sdk;
    private readonly Dictionary<string, Payload> _payloadCache = new();

    public LockboxSecretConfigurationProvider(IConfigurationRoot innerConfig, Sdk sdk)
    {
        _innerConfig = innerConfig;
        _sdk = sdk;
    }

    public override void Load()
    {
        // Collect all key-value pairs from the inner configuration that need resolution.
        var secretRefs = new List<(string ConfigKey, string SecretSpec)>();

        foreach (var (key, value) in _innerConfig.AsEnumerable())
        {
            if (value is not null && value.StartsWith(SecretPrefix, StringComparison.Ordinal))
            {
                secretRefs.Add((key, value));
            }
        }

        if (secretRefs.Count == 0)
            return;

        // Resolve all secrets synchronously (configuration providers load synchronously in .NET).
        foreach (var (configKey, secretSpec) in secretRefs)
        {
            var resolved = ResolveSecret(secretSpec);
            Data[configKey] = resolved;
        }
    }

    private string ResolveSecret(string secretSpec)
    {
        var (secretId, versionId, key) = ParseSecretSpec(secretSpec);

        var cacheKey = $"{secretId}:{versionId}";

        if (!_payloadCache.TryGetValue(cacheKey, out var payload))
        {
            payload = _sdk.Services.Lockbox.PayloadService
                .Get(new GetPayloadRequest
                {
                    SecretId = secretId,
                    VersionId = versionId,
                });

            _payloadCache[cacheKey] = payload;
        }

        var entry = payload.Entries.FirstOrDefault(e => e.Key == key);
        if (entry is null)
        {
            throw new InvalidOperationException(
                $"Key '{key}' not found in Lockbox secret '{secretId}' (version '{versionId}'). " +
                $"Available keys: [{string.Join(", ", payload.Entries.Select(e => e.Key))}]");
        }

        return entry.TextValue;
    }

    /// <summary>
    /// Parses a secret specification string into its components.
    /// Expected format: <c>secret:{id}:{version}:{key}</c>
    /// </summary>
    public static (string SecretId, string VersionId, string Key) ParseSecretSpec(string secretSpec)
    {
        var parts = secretSpec.Split(':', 4);

        if (parts.Length != 4 || parts[0] != "secret")
        {
            throw new FormatException(
                $"Invalid secret format. Expected 'secret:{{id}}:{{version}}:{{key}}', got '{secretSpec}'");
        }

        var secretId = parts[1];
        var versionId = parts[2];
        var key = parts[3];

        if (string.IsNullOrWhiteSpace(secretId))
            throw new FormatException($"Secret ID cannot be empty in '{secretSpec}'");

        if (string.IsNullOrWhiteSpace(versionId))
            throw new FormatException($"Version ID cannot be empty in '{secretSpec}'");

        if (string.IsNullOrWhiteSpace(key))
            throw new FormatException($"Key cannot be empty in '{secretSpec}'");

        return (secretId, versionId, key);
    }
}
