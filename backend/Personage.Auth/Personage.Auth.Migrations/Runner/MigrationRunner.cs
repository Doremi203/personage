using System.Reflection;
using Dapper;
using Microsoft.Extensions.Configuration;
using Microsoft.Extensions.Logging;
using Npgsql;
using Personage.Auth.Migrations.Runner.Models;

namespace Personage.Auth.Migrations.Runner;

public interface IMigrationRunner
{
    Task RunMigrations();
}

public class MigrationRunner(
    IConfiguration configuration,
    ILogger<MigrationRunner> logger
) : IMigrationRunner
{
    private readonly string _connectionString =
        configuration.GetConnectionString("AuthDb")
        ?? throw new ArgumentNullException(nameof(configuration));

    public async Task RunMigrations()
    {
        logger.LogInformation("Starting database migrations...");

        await using var connection = new NpgsqlConnection(_connectionString);
        await connection.OpenAsync();
         
        await EnsureMigrationsTableExists(connection);
        var migrationFiles = await GetMigrationFiles();
        var appliedMigrations = await GetAppliedMigrations(connection);
        
        foreach (var migration in migrationFiles.OrderBy(m => m.Name))
        {
            if (!appliedMigrations.Contains(migration.Name))
            {
                logger.LogInformation("Applying migration: {MigrationName}", migration.Name);
                await ApplyMigration(connection, migration);
                logger.LogInformation("Applied migration: {MigrationName}", migration.Name);
            }
            else
            {
                logger.LogDebug("Migration already applied: {MigrationName}", migration.Name);
            }
        }
        
        logger.LogInformation("Database migrations completed successfully");
    }
    
    private async Task EnsureMigrationsTableExists(NpgsqlConnection connection)
    {
        try
        {
            await connection.ExecuteAsync(
                """
                CREATE TABLE IF NOT EXISTS migrations (
                    id SERIAL PRIMARY KEY,
                    name VARCHAR(255) UNIQUE NOT NULL,
                    applied_at TIMESTAMPTZ DEFAULT NOW()
                )
                """);
        }
        catch (Exception ex)
        {
            logger.LogWarning(ex, "Could not ensure migrations table exists. It may already exist.");
        }
    }
    
    private async Task<List<MigrationFile>> GetMigrationFiles()
    {
        var migrations = new List<MigrationFile>();
        var assembly = Assembly.GetExecutingAssembly();
        var resourceNames = assembly
            .GetManifestResourceNames()
            .Where(name => name.EndsWith(".sql", StringComparison.OrdinalIgnoreCase))
            .OrderBy(name => name)
            .ToList();
        
        if (resourceNames.Count == 0)
        {
            logger.LogError("No SQL migration files found in assembly");
            return migrations;
        }
        
        foreach (var resourceName in resourceNames)
        {
            await using var stream = assembly.GetManifestResourceStream(resourceName);
            if (stream == null) continue;
            
            using var reader = new StreamReader(stream);
            var content = await reader.ReadToEndAsync();

            var migrationName =
                resourceName
                    .Split('.')
                    .TakeLast(2)
                    .First();
            migrations.Add(new MigrationFile
            {
                Name = migrationName,
                Content = content
            });
            
            logger.LogDebug("Loaded migration: {MigrationName}", migrationName);
        }
        
        logger.LogInformation("Found {Count} migration files", migrations.Count);
        return migrations;
    }
    
    private async Task<HashSet<string>> GetAppliedMigrations(NpgsqlConnection connection)
    {
        try
        {
            var migrations = await connection.QueryAsync<string>(
                "SELECT name FROM migrations ORDER BY applied_at");
            return migrations.ToHashSet();
        }
        catch (Exception ex)
        {
            logger.LogWarning(ex, "Could not query applied migrations. Returning empty set.");
            return [];
        }
    }
    
    private async Task ApplyMigration(NpgsqlConnection connection, MigrationFile migration)
    {
        await using var transaction = await connection.BeginTransactionAsync();
        
        try
        {
            await connection.ExecuteAsync(migration.Content, transaction: transaction);
            
            await connection.ExecuteAsync(
                "INSERT INTO migrations (name) VALUES (@name)",
                new { name = migration.Name },
                transaction: transaction);
            
            await transaction.CommitAsync();
        }
        catch (Exception ex)
        {
            await transaction.RollbackAsync();
            logger.LogError(ex, "Failed to apply migration: {MigrationName}", migration.Name);
            throw;
        }
    }
}