using System.Security.Cryptography;
using DotNetEnv;
using Npgsql;
using Microsoft.Extensions.Logging.Console;
using Microsoft.Extensions.Options;
using Microsoft.IdentityModel.Tokens;
using Personage.Auth.Api.Configuration;
using Personage.Auth.Api.Contracts.Common;
using Personage.Auth.Api.GrpcClients;
using Personage.Auth.Api.GrpcServices;
using Personage.Auth.Api.Logging;
using Personage.Auth.Api.Middleware;
using Personage.Auth.Api.Middleware.Rest;
using Personage.Auth.Bll.Services;
using TelegramChatsGrpcServiceClient = Personage.Auth.Api.Grpc.TelegramChats.TelegramChatsService.TelegramChatsServiceClient;
using Personage.Auth.DataAccess;
using Personage.Auth.DataAccess.Interfaces;
using Personage.Auth.DataAccess.Interfaces.Repositories;
using Personage.Auth.DataAccess.Repositories;
using Personage.Auth.Domain.Configuration;
using Personage.Auth.Domain.Interfaces;
using Microsoft.AspNetCore.Authentication.JwtBearer;


namespace Personage.Auth.Api;

public class Program
{
    public static async Task Main(string[] args)
    {
        var builder = WebApplication.CreateBuilder(args);
        builder.WebHost.ConfigureKestrel(serverOptions =>
        {
            serverOptions.AllowAlternateSchemes = true;
        });

        if (builder.Environment.IsProduction())
        {
            builder.Logging.AddConsoleFormatter<CompactJsonConsoleFormatter, ConsoleFormatterOptions>();
            builder.Logging.AddConsole(options =>
            {
                options.FormatterName = CompactJsonConsoleFormatter.FormatterName;
            });
        }

        var useMetadataService = builder.Configuration.GetValue<bool>("YandexCloud:UseMetadataService");
        builder.Configuration.AddLockboxSecrets(useMetadataService);

        builder.Services.AddGrpc(options =>
        {
            options.EnableDetailedErrors = true;
            options.Interceptors.Add<ExceptionInterceptor>();
            options.IgnoreUnknownServices = true;
        });

        builder.Services.AddGrpcReflection();

        var services = builder.Services;
        var configuration = builder.Configuration;
        var environment = builder.Environment;

        ConfigureServices(services, configuration, environment);

        var app = builder.Build();

        // Merge the resolved Password into the ConnectionString if present.
        MergeDatabasePassword(app);

        ConfigureMiddleware(app);

        await app.RunAsync();
    }

    /// <summary>
    /// If <see cref="ConnectionFactorySettings.Password"/> is set (e.g., resolved from Lockbox),
    /// merge it into the connection string. This keeps the config format consistent with
    /// Go/traitex services where the password is a separate field.
    /// </summary>
    private static void MergeDatabasePassword(WebApplication app)
    {
        var settings = app.Services.GetRequiredService<IOptions<ConnectionFactorySettings>>().Value;
        if (string.IsNullOrEmpty(settings.Password))
            return;

        var connBuilder = new NpgsqlConnectionStringBuilder(settings.ConnectionString)
        {
            Password = settings.Password
        };
        settings.ConnectionString = connBuilder.ConnectionString;

        var logger = app.Services.GetRequiredService<ILogger<Program>>();
        logger.LogInformation("Database password merged into connection string from configuration");
    }

    private static void ConfigureServices(
        IServiceCollection services,
        ConfigurationManager configuration,
        IWebHostEnvironment environment)
    {
        services.AddScoped<IDbConnectionFactory, DbConnectionFactory>();
        services.AddSingleton<ExceptionInterceptor>();

        services.AddHttpContextAccessor();

        ConfigureSettings(services, configuration, environment);
        AddRepositories(services);
        AddBllServices(services);

        AddReflectionAndSwagger(services);
        AddAuthentication(services, configuration);
        AddCors(services, configuration);

        services.AddControllers();
    }

    private static void ConfigureMiddleware(WebApplication app)
    {
        app.UseMiddleware<ExceptionHandlingMiddleware>();

        app.UseSwagger();
        app.UseSwaggerUI(c =>
        {
            c.SwaggerEndpoint("/swagger/v1/swagger.json", "Personage.Auth API v1");
            c.DisplayOperationId();
            c.DisplayRequestDuration();
        });

        app.UseCors();
        app.UseAuthentication();
        app.UseAuthorization();

        app.UseGrpcWeb();
        app.MapGrpcReflectionService();
        app.MapGrpcService<AuthGrpcService>().EnableGrpcWeb();
        app.MapGrpcService<StateTrackingGrpcService>().EnableGrpcWeb();
        app.MapGrpcService<TelegramGrpcService>().EnableGrpcWeb();

        app.MapControllers();
    }

    private static void AddAuthentication(
        IServiceCollection services,
        IConfiguration configuration
        )
    {
        services.AddAuthentication(JwtBearerDefaults.AuthenticationScheme)
            .AddJwtBearer(options =>
            {
                var jwtSettings = configuration.GetSection(nameof(JwtSettings)).Get<JwtSettings>();

                options.TokenValidationParameters = new TokenValidationParameters
                {
                    ValidateIssuer = true,
                    ValidIssuer = jwtSettings!.Issuer,
                    ValidateAudience = true,
                    ValidAudience = jwtSettings.Audience,
                    ValidateLifetime = true,
                    ValidateIssuerSigningKey = true,
                    IssuerSigningKey = CreateRsaSecurityKeyFromPublicKeyPem(jwtSettings.PublicKeyPem)
                };
            });

        services.AddAuthorization();
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

            c.AddSecurityDefinition("Bearer", new Microsoft.OpenApi.Models.OpenApiSecurityScheme
            {
                Description = "JWT Authorization header using the Bearer scheme. Enter 'Bearer' [space] and then your token",
                Name = "Authorization",
                In = Microsoft.OpenApi.Models.ParameterLocation.Header,
                Type = Microsoft.OpenApi.Models.SecuritySchemeType.ApiKey,
                Scheme = "Bearer"
            });

            c.AddSecurityRequirement(new Microsoft.OpenApi.Models.OpenApiSecurityRequirement
            {
                {
                    new Microsoft.OpenApi.Models.OpenApiSecurityScheme
                    {
                        Reference = new Microsoft.OpenApi.Models.OpenApiReference
                        {
                            Type = Microsoft.OpenApi.Models.ReferenceType.SecurityScheme,
                            Id = "Bearer"
                        }
                    },
                    Array.Empty<string>()
                }
            });

            c.MapType<ErrorResponse>(() => new Microsoft.OpenApi.Models.OpenApiSchema
            {
                Type = "object",
                Properties = new Dictionary<string, Microsoft.OpenApi.Models.OpenApiSchema>
                {
                    ["errorCode"] = new() { Type = "string" },
                    ["message"] = new() { Type = "string" },
                    ["statusCode"] = new() { Type = "integer" }
                }
            });
        });
    }

    private static void AddRepositories(IServiceCollection services)
    {
        services.AddScoped<IUserRepository, UserRepository>();
        services.AddScoped<IGmailTokenRepository, GmailTokenRepository>();
        services.AddScoped<IGoogleCalendarTokenRepository, GoogleCalendarTokenRepository>();
        services.AddScoped<IOAuthStateRepository, OAuthStateRepository>();
        services.AddScoped<IRefreshTokenRepository, RefreshTokenRepository>();
        services.AddScoped<IPasswordResetTokenRepository, PasswordResetTokenRepository>();
        services.AddScoped<ITelegramSessionRepository, TelegramSessionRepository>();
        services.AddScoped<ITelegramChatRepository, TelegramChatRepository>();
    }

    private static void AddBllServices(IServiceCollection services)
    {
        services.AddScoped<IAuthService, AuthService>();
        services.AddScoped<ITokenService, TokenService>();
        services.AddScoped<IStateTrackingService, StateTrackingService>();
        services.AddScoped<IGoogleOAuthService, GoogleOAuthService>();
        services.AddScoped<IPostboxService, PostboxService>();
        services.AddScoped<ITelegramAuthService, TelegramAuthService>();
        services.AddScoped<ITelegramChatsService, TelegramChatsService>();
        services.AddScoped<IClaimValues, ClaimValues>();
        services.AddScoped<IUserService, UserService>();
        services.AddScoped<IUserProfileService, UserProfileService>();
        services.AddScoped<IIntegrationsService, IntegrationsService>();
        services.AddHttpClient<IGoogleOAuthService, GoogleOAuthService>();
        services.AddScoped<ITelegramChatsGrpcClient, TelegramChatsGrpcClient>();
        services
            .AddGrpcClient<TelegramChatsGrpcServiceClient>((sp, options) =>
            {
                var grpcSettings = sp.GetRequiredService<IOptions<TelegramAuthGrpcSettings>>().Value;
                options.Address = new Uri(grpcSettings.Url);
            });
    }

    private static void ConfigureSettings(
        IServiceCollection services,
        ConfigurationManager configuration,
        IWebHostEnvironment environment)
    {
        services.Configure<OAuthSettings>(configuration.GetSection(nameof(OAuthSettings)));

        var clientId = configuration["OAuthSettings:ClientId"];
        var clientSecret = configuration["OAuthSettings:ClientSecret"];
        var scopes = configuration.GetSection("OAuthSettings:Scopes").Get<string[]>();

        if (!environment.IsProduction())
        {
            Env.Load();
            clientId ??= Environment.GetEnvironmentVariable("OAUTH_CLIENT_ID");
            clientSecret ??= Environment.GetEnvironmentVariable("OAUTH_CLIENT_SECRET");
            scopes ??= Environment.GetEnvironmentVariable("OAUTH_SCOPES")?.Split(',');

            services.Configure<OAuthSettings>(options =>
            {
                options.ClientId = clientId ?? throw new InvalidOperationException("OAUTH_CLIENT_ID not set");
                options.ClientSecret = clientSecret ?? throw new InvalidOperationException("OAUTH_CLIENT_SECRET not set");
                options.Scopes = scopes ?? throw new InvalidOperationException("OAUTH_SCOPES not set");
            });
        }

        services.Configure<ConnectionFactorySettings>(configuration.GetSection(nameof(ConnectionFactorySettings)));
        services.Configure<JwtSettings>(configuration.GetSection(nameof(JwtSettings)));
        services.Configure<PostboxSettings>(configuration.GetSection(nameof(PostboxSettings)));
        services.Configure<AdminSettings>(configuration.GetSection(nameof(AdminSettings)));
        services.Configure<TelegramAuthGrpcSettings>(configuration.GetSection(nameof(TelegramAuthGrpcSettings)));
    }

    private static void AddCors(IServiceCollection services, ConfigurationManager configuration)
    {
        var corsSettings = configuration.GetSection("Cors");
        var allowedOrigins = corsSettings.GetSection("AllowedOrigins").Get<string[]>() ?? [];
        var allowedMethods = corsSettings.GetSection("AllowedMethods").Get<string[]>() ??
        [
            "GET", "POST", "PUT", "DELETE", "OPTIONS"
        ];
        var allowedHeaders = corsSettings.GetSection("AllowedHeaders").Get<string[]>() ??
        [
            "Content-Type", "Authorization"
        ];
        var exposedHeaders = corsSettings.GetSection("ExposedHeaders").Get<string[]>() ?? [];

        services.AddCors(options =>
        {
            options.AddDefaultPolicy(policy =>
            {
                if (allowedOrigins.Length > 0)
                {
                    policy.WithOrigins(allowedOrigins)
                          .WithMethods(allowedMethods)
                          .WithHeaders(allowedHeaders)
                          .AllowCredentials();

                    if (exposedHeaders.Length > 0)
                    {
                        policy.WithExposedHeaders(exposedHeaders);
                    }
                }
                else
                {
                    policy.AllowAnyOrigin()
                          .AllowAnyMethod()
                          .AllowAnyHeader();
                }
            });
        });
    }

    private static RsaSecurityKey CreateRsaSecurityKeyFromPublicKeyPem(string publicKeyPem)
    {
        if (string.IsNullOrWhiteSpace(publicKeyPem))
            throw new InvalidOperationException("Public key PEM is not configured");

        var pem = publicKeyPem
            .Replace("-----BEGIN PUBLIC KEY-----", "")
            .Replace("-----END PUBLIC KEY-----", "")
            .Replace("\n", "")
            .Replace("\r", "")
            .Trim();

        var keyBytes = Convert.FromBase64String(pem);

        var rsa = RSA.Create();
        rsa.ImportSubjectPublicKeyInfo(keyBytes, out _);

        return new RsaSecurityKey(rsa);
    }
}
