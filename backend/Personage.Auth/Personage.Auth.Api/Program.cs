using Personage.Auth.DataAccess;
using Personage.Auth.DataAccess.Interfaces;
using Personage.Auth.DataAccess.Interfaces.Repositories;
using Personage.Auth.DataAccess.Repositories;
using Personage.Auth.Domain.Configuration;
using Personage.Auth.Domain.Interfaces;
using Personage.Auth.Domain.Services;
using Personage.Auth.GrpcServices;
using Personage.Auth.Migrations.Runner;

namespace Personage.Auth;

public class Program
{
    public static async Task Main(string[] args)
    {
        var builder = WebApplication.CreateBuilder(args);
        builder.Services.AddGrpc(options =>
        {
            options.EnableDetailedErrors = true;
        });

        var services = builder.Services;
        var configuration = builder.Configuration;
        services.AddScoped<IDbConnectionFactory, DbConnectionFactory>();
        
        ConfigureSettings(services, configuration);
        AddRepositories(services);
        AddBllServices(services);
        services.AddScoped<IMigrationRunner, MigrationRunner>();
        
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
        services.AddControllers();
        var app = builder.Build();
        var logger = app.Logger;
        
        if (args.Contains("migrate"))
        {
            logger.LogInformation("Running database migrations...");
    
            using var scope = app.Services.CreateScope();
            var migrationRunner = scope.ServiceProvider.GetRequiredService<IMigrationRunner>();
    
            try
            {
                await migrationRunner.RunMigrations();
                logger.LogInformation("Migrations completed successfully!");
                return;
            }
            catch (Exception ex)
            {
                logger.LogError(ex, "Migrations failed");
                return;
            }
        }

        app.UseSwagger();
        app.UseSwaggerUI(c =>
        {
            c.SwaggerEndpoint("/swagger/v1/swagger.json", "Personage.Auth API v1");
            
            c.DisplayOperationId();
            c.DisplayRequestDuration();
        });
        
        app.MapGrpcReflectionService();
        app.MapControllers();
        
        app.UseGrpcWeb();
        app.MapGrpcService<TestGrpcService>().EnableGrpcWeb();
        
        await app.RunAsync();
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
        services.AddHttpClient<GoogleOAuthService>();
    }

    private static void ConfigureSettings(IServiceCollection services, IConfiguration configuration)
    {
        services.Configure<OAuthSettings>(configuration.GetSection("OAuthSettings:Gmail"));
    }
}