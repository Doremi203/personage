using DotNetEnv;
using Personage.Auth.Api.GrpcServices;
using Personage.Auth.Api.Middleware;
using Personage.Auth.Bll.Services;
using Personage.Auth.DataAccess;
using Personage.Auth.DataAccess.Interfaces;
using Personage.Auth.DataAccess.Interfaces.Repositories;
using Personage.Auth.DataAccess.Repositories;
using Personage.Auth.Domain.Configuration;
using Personage.Auth.Domain.Interfaces;
using Personage.Auth.Migrations.Runner;

namespace Personage.Auth.Api;

public class Program
{
    public static async Task Main(string[] args)
    {
        var builder = WebApplication.CreateBuilder(args);
        builder.Services.AddGrpc(options =>
        {
            options.EnableDetailedErrors = true;
            options.Interceptors.Add<ExceptionInterceptor>();
            options.IgnoreUnknownServices = true;
        });
        
        builder.WebHost.ConfigureKestrel(options =>
        {
            options.ConfigureEndpointDefaults(defaultOptions =>
            {
                defaultOptions.Protocols = Microsoft.AspNetCore.Server.Kestrel.Core.HttpProtocols.Http1AndHttp2;
            });
        });
        
        builder.Services.AddGrpcReflection();

        var services = builder.Services;
        var configuration = builder.Configuration;
        var environment = builder.Environment;
        
        ConfigureServices(services, configuration, environment);
        
        var app = builder.Build();
        
        if (args.Contains("migrate"))
        {
            await MigrateDatabase(app);
            return;
        }

        ConfigureMiddleware(app);
        
        await app.RunAsync();
    }

    private static void ConfigureServices(
        IServiceCollection services, 
        ConfigurationManager configuration, 
        IWebHostEnvironment environment)
    {
        services.AddScoped<IDbConnectionFactory, DbConnectionFactory>();
        services.AddSingleton<ExceptionInterceptor>();
        
        ConfigureSettings(services, configuration, environment);
        AddRepositories(services);
        AddBllServices(services);
        AddReflectionAndSwagger(services);
        
        services.AddScoped<IMigrationRunner, MigrationRunner>();
        services.AddControllers();
    }

    private static async Task MigrateDatabase(WebApplication app)
    {
        var logger = app.Logger;
        logger.LogInformation("Running database migrations...");
    
        using var scope = app.Services.CreateScope();
        var migrationRunner = scope.ServiceProvider.GetRequiredService<IMigrationRunner>();
    
        try
        {
            await migrationRunner.RunMigrations();
            logger.LogInformation("Migrations completed successfully!");
        }
        catch (Exception ex)
        {
            logger.LogError(ex, "Migrations failed");
            throw;
        }
    }
    
    private static void ConfigureMiddleware(WebApplication app)
    {
        app.UseSwagger();
        app.UseSwaggerUI(c =>
        {
            c.SwaggerEndpoint("/swagger/v1/swagger.json", "Personage.Auth API v1");
            c.DisplayOperationId();
            c.DisplayRequestDuration();
        });
        

        app.UseGrpcWeb();
        app.MapGrpcReflectionService();
        app.MapGrpcService<TestGrpcService>().EnableGrpcWeb();
        app.MapGrpcService<AuthGrpcService>().EnableGrpcWeb();
        
        app.MapControllers();
    }
    
    private static void AddReflectionAndSwagger(IServiceCollection services)
    {
        services.AddGrpcReflection();
        services.AddEndpointsApiExplorer();
        services.AddSwaggerGen(c =>
        {
            c.SwaggerDoc("v1", new Microsoft.OpenApi.Models.OpenApiInfo
            {
                Title = "Personage.Auth API",
                Version = "v1",
                Description = "Authentication API with gRPC and REST endpoints"
            });
        });
    }

    private static void AddRepositories(IServiceCollection services)
    {
        services.AddScoped<IUserRepository, UserRepository>();
        services.AddScoped<IGmailTokenRepository, GmailTokenRepository>();
        services.AddScoped<IOAuthStateRepository, OAuthStateRepository>();
    }

    private static void AddBllServices(IServiceCollection services)
    {
        services.AddScoped<IAuthService, AuthService>();
        services.AddScoped<IGoogleOAuthService, GoogleOAuthService>();
        services.AddHttpClient<IGoogleOAuthService, GoogleOAuthService>();
    }

    private static void ConfigureSettings(
        IServiceCollection services,
        ConfigurationManager configuration,
        IWebHostEnvironment environment)
    {
        services.Configure<OAuthSettings>(configuration.GetSection(nameof(OAuthSettings)));
        if (!environment.IsProduction())
        {
            Env.Load();
            var clientId = Environment.GetEnvironmentVariable("OAUTH_CLIENT_ID");
            var clientSecret = Environment.GetEnvironmentVariable("OAUTH_CLIENT_SECRET");
            var scopes = Environment.GetEnvironmentVariable("OAUTH_SCOPES")?.Split(',');
            
            services.Configure<OAuthSettings>(options =>
            {
                options.ClientId = clientId ?? throw new InvalidOperationException("OAUTH_CLIENT_ID not set");
                options.ClientSecret = clientSecret ?? throw new InvalidOperationException("OAUTH_CLIENT_SECRET not set");
                options.Scopes = scopes ?? throw new InvalidOperationException("OAUTH_SCOPES not set");
            });
        }
        
        services.Configure<ConnectionFactorySettings>(configuration.GetSection(nameof(ConnectionFactorySettings)));
    }
}