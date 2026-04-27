using System;
using System.Diagnostics;
using System.IO;
using System.Linq;
using System.Net.Http;
using Microsoft.AspNetCore.Hosting;
using Microsoft.AspNetCore.Mvc.Testing;
using Microsoft.AspNetCore.TestHost;
using Microsoft.Extensions.Configuration;
using Microsoft.Extensions.DependencyInjection;
using Moq;
using Personage.Auth.Api;
using Personage.Auth.Bll.Services;
using Personage.Auth.Domain.Interfaces;
using Personage.Auth.Tests.Infrastructure.Repositories;

namespace Personage.Auth.Tests.Infrastructure;

public sealed class TestApplicationFactory(Action<IServiceCollection> overrideServices) : WebApplicationFactory<Program>
{
    public Mock<HttpMessageHandler> HttpMessageHandlerMock { get; } = new();
    private bool _isMigrationsApplied;
    private readonly object _lock = new();

    protected override void ConfigureWebHost(IWebHostBuilder builder)
    {
        builder.UseEnvironment("Testing");

        builder.ConfigureTestServices(services =>
        {
            var descriptor = services.FirstOrDefault(
                d => d.ServiceType == typeof(IHttpClientFactory) ||
                     (d.ServiceType.IsGenericType &&
                      d.ServiceType.GetGenericTypeDefinition() == typeof(IHttpClientFactory)));

            if (descriptor != null) services.Remove(descriptor);

            services.AddHttpClient<IGoogleOAuthService, GoogleOAuthService>()
                .ConfigurePrimaryHttpMessageHandler(() => HttpMessageHandlerMock.Object);

            services.AddScoped<TestCleaners>();
            services.AddScoped<TestOAuthStateRepository>();
            services.AddScoped<TestUserRepository>();

            overrideServices.Invoke(services);
        });
    }

    protected override void ConfigureClient(HttpClient client)
    {
        base.ConfigureClient(client);

        lock (_lock)
        {
            if (_isMigrationsApplied) return;

            RunGooseMigrations();
            _isMigrationsApplied = true;
        }
    }

    private void RunGooseMigrations()
    {
        var configuration = Services.GetRequiredService<IConfiguration>();
        var connectionString = configuration["ConnectionFactorySettings:ConnectionString"]
                               ?? throw new InvalidOperationException("ConnectionFactorySettings:ConnectionString is not configured");

        // Convert ADO.NET connection string to goose-compatible PostgreSQL URI
        var npgsqlBuilder = new Npgsql.NpgsqlConnectionStringBuilder(connectionString);
        var gooseDbString = $"postgres://{npgsqlBuilder.Username}:{npgsqlBuilder.Password}" +
                            $"@{npgsqlBuilder.Host}:{npgsqlBuilder.Port}/{npgsqlBuilder.Database}?sslmode=disable";

        // Resolve migrations directory relative to the test assembly output directory.
        // Test output is typically in: Personage.Auth.Tests/bin/Debug/net9.0/
        // Migrations are in:           Personage.Auth/migrations/
        var baseDir = AppContext.BaseDirectory;
        var migrationsDir = Path.GetFullPath(Path.Combine(baseDir, "../../../../migrations"));

        if (!Directory.Exists(migrationsDir))
            throw new DirectoryNotFoundException($"Migrations directory not found: {migrationsDir}");

        var process = new Process
        {
            StartInfo = new ProcessStartInfo
            {
                FileName = "goose",
                Arguments = $"-dir \"{migrationsDir}\" postgres \"{gooseDbString}\" up",
                RedirectStandardOutput = true,
                RedirectStandardError = true,
                UseShellExecute = false,
                CreateNoWindow = true,
            }
        };

        process.Start();
        var stdout = process.StandardOutput.ReadToEnd();
        var stderr = process.StandardError.ReadToEnd();
        process.WaitForExit();

        if (process.ExitCode != 0)
            throw new InvalidOperationException(
                $"goose migrations failed (exit code {process.ExitCode}):\n{stderr}\n{stdout}");
    }
}
