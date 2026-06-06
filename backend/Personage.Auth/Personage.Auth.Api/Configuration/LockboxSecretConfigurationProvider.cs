using Grpc.Core;
using Grpc.Net.Client;
using Yandex.Cloud.Credentials;
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
///
/// The Lockbox call is made over a gRPC channel built here directly, rather than
/// through <c>Yandex.Cloud.Sdk</c>. The SDK creates its channel without an explicit
/// <c>HttpClient</c>, which routes calls through the gRPC client-side load balancer
/// (<c>BalancerHttpHandler</c>) added in Grpc.Net.Client 2.44+. That balancer makes
/// the call to Yandex Cloud fail with an HTTP/2 <c>COMPRESSION_ERROR</c>
/// (see yandex-cloud/dotnet-sdk#33 and grpc/grpc-dotnet#2254). Supplying an explicit
/// <c>HttpClient</c> disables load balancing and uses a plain HTTP/2 connection,
/// which works. We also talk to the Lockbox payload endpoint directly, skipping the
/// SDK's endpoint-discovery round-trip to <c>api.cloud.yandex.net</c> (the exact call
/// that crashed on startup).
/// </summary>
public sealed class LockboxSecretConfigurationProvider : ConfigurationProvider
{
    public const string SecretPrefix = "secret:";

    private readonly IConfigurationRoot _innerConfig;
    private readonly ICredentialsProvider _credentialsProvider;
    private readonly string _payloadEndpoint;
    private readonly Dictionary<string, Payload> _payloadCache = new();

    public LockboxSecretConfigurationProvider(
        IConfigurationRoot innerConfig,
        ICredentialsProvider credentialsProvider,
        string payloadEndpoint)
    {
        _innerConfig = innerConfig;
        _credentialsProvider = credentialsProvider;
        _payloadEndpoint = payloadEndpoint;
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

        // Explicit HttpClient => gRPC client-side load balancing is disabled (plain HTTP/2),
        // avoiding the BalancerHttpHandler path that fails with COMPRESSION_ERROR.
        using var httpClient = new HttpClient();
        using var channel = GrpcChannel.ForAddress(
            $"https://{_payloadEndpoint}",
            new GrpcChannelOptions
            {
                HttpClient = httpClient,
                DisposeHttpClient = false,
            });

        var client = new PayloadService.PayloadServiceClient(channel);

        // Resolve all secrets synchronously (configuration providers load synchronously in .NET).
        foreach (var (configKey, secretSpec) in secretRefs)
        {
            Data[configKey] = ResolveSecret(client, secretSpec);
        }
    }

    private string ResolveSecret(PayloadService.PayloadServiceClient client, string secretSpec)
    {
        var (secretId, versionId, key) = ParseSecretSpec(secretSpec);

        var cacheKey = $"{secretId}:{versionId}";

        if (!_payloadCache.TryGetValue(cacheKey, out var payload))
        {
            var headers = new Metadata
            {
                { "authorization", $"Bearer {_credentialsProvider.GetToken()}" },
            };

            payload = client.Get(
                new GetPayloadRequest
                {
                    SecretId = secretId,
                    VersionId = versionId,
                },
                headers);

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
