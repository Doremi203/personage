using Yandex.Cloud.Credentials;

namespace Personage.Auth.Api.Extensions;

/// <summary>
/// Simple IAM token credentials provider for local development.
/// Used by <see cref="Configuration.LockboxConfigurationExtensions"/> when
/// <c>YandexCloud:UseMetadataService</c> is <c>false</c>.
/// </summary>
public class IamTokenCredentialsProvider(string iamToken) : ICredentialsProvider
{
    public string GetToken()
    {
        return iamToken;
    }
}
