using DotNetEnv;
using Yandex.Cloud;
using Yandex.Cloud.Credentials;

namespace Personage.Auth.Api.Extensions;

public class IamTokenCredentialsProvider(string iamToken) : ICredentialsProvider
{
    public string GetToken()
    {
        return iamToken;
    }
}

public static class YandexServicesExtensions
{
    private const string LocalSecretsPath = "../../../secrets.env";
    private const string TokenEnvVariableName = "YC_TOKEN";
    public static void AddYandexCloudSdk(this IServiceCollection services, IConfiguration configuration)
    {
        var useMetadataService = configuration.GetValue<bool>("YandexCloud:UseMetadataService");
        
        if (useMetadataService)
        {
            services.AddScoped<Sdk>(_ => new Sdk(new MetadataCredentialsProvider()));
        }
        else
        {
            var secretsPath = Path.Combine(Directory.GetCurrentDirectory(), LocalSecretsPath);
            if (File.Exists(secretsPath))
                Env.Load(secretsPath);
            else
                Env.Load();
            
            var iamToken = Environment.GetEnvironmentVariable(TokenEnvVariableName) 
                           ?? throw new InvalidOperationException($"{TokenEnvVariableName} environment variable is not set");
            
            
            services.AddScoped<Sdk>(_ => new Sdk(new IamTokenCredentialsProvider(iamToken)));
        }
    }
}