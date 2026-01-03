using System.Security.Cryptography;
using System.Text;
using Dapper;
using Microsoft.Extensions.Configuration;
using Microsoft.Extensions.Logging;
using Npgsql;

namespace Personage.Auth.Migrations;

public interface IMigrationRunner
{
    Task RunMigrations();
}

public class MigrationRunner(IConfiguration configuration, ILogger<MigrationRunner> logger)
    : IMigrationRunner
{
    private readonly string _connectionString =
        configuration.GetConnectionString("AuthDb")
        ?? throw new ArgumentNullException(nameof(configuration));

    public async Task RunMigrations()
    {
        logger.LogInformation("Starting database migrations...");

        await using var connection = new NpgsqlConnection(_connectionString);
        await connection.OpenAsync();
         
        await EnsureMigrationsTableExistsAsync(connection);
        var migrationFiles = await GetMigrationFilesAsync();
        
        // Get already applied migrations
        var appliedMigrations = await GetAppliedMigrationsAsync(connection);
        
        // Apply migrations in order
        foreach (var migration in migrationFiles.OrderBy(m => m.Name))
        {
            if (!appliedMigrations.Contains(migration.Name))
            {
                logger.LogInformation("Applying migration: {MigrationName}", migration.Name);
                await ApplyMigrationAsync(connection, migration);
                logger.LogInformation("Applied migration: {MigrationName}", migration.Name);
            }
            else
            {
                logger.LogDebug("Migration already applied: {MigrationName}", migration.Name);
            }
        }
        
        logger.LogInformation("Database migrations completed successfully");
    }
    
    private async Task EnsureMigrationsTableExistsAsync(NpgsqlConnection connection)
    {
        try
        {
            await connection.ExecuteAsync(
                """
                CREATE TABLE IF NOT EXISTS migrations (
                    id SERIAL PRIMARY KEY,
                    name VARCHAR(255) UNIQUE NOT NULL,
                    applied_at TIMESTAMPTZ DEFAULT NOW(),
                    checksum VARCHAR(64)
                )
                """);
        }
        catch (Exception ex)
        {
            logger.LogWarning(ex, "Could not ensure migrations table exists. It may already exist.");
        }
    }
    
    private async Task<List<MigrationFile>> GetMigrationFilesAsync()
    {
        var migrations = new List<MigrationFile>();
        var migrationsPath = Directory.GetCurrentDirectory();
        
        if (!Directory.Exists(migrationsPath))
        {
            logger.LogError("Migrations directory not found: {MigrationsPath}", migrationsPath);
            return migrations;
        }
        
        var files = Directory.GetFiles(migrationsPath, "*.sql")
            .OrderBy(Path.GetFileName)
            .ToList();
        
        foreach (var filePath in files)
        {
            var fileName = Path.GetFileName(filePath);
            var content = await File.ReadAllTextAsync(filePath);
            var checksum = CalculateChecksum(content);
            
            migrations.Add(new MigrationFile
            {
                Name = fileName,
                Path = filePath,
                Content = content,
                Checksum = checksum
            });
        }
        
        return migrations;
    }
    
    private async Task<HashSet<string>> GetAppliedMigrationsAsync(NpgsqlConnection connection)
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
            return new HashSet<string>();
        }
    }
    
    private async Task ApplyMigrationAsync(NpgsqlConnection connection, MigrationFile migration)
    {
        using var transaction = await connection.BeginTransactionAsync();
        
        try
        {
            // Execute the migration SQL
            await connection.ExecuteAsync(migration.Content, transaction: transaction);
            
            // Record the migration
            await connection.ExecuteAsync(
                @"INSERT INTO migrations (name, checksum) VALUES (@name, @checksum)",
                new { name = migration.Name, checksum = migration.Checksum },
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
    
    private string CalculateChecksum(string content)
    {
        using var sha256 = SHA256.Create();
        var bytes = Encoding.UTF8.GetBytes(content);
        var hash = sha256.ComputeHash(bytes);
        return BitConverter.ToString(hash).Replace("-", "").ToLower();
    }
    
    private class MigrationFile
    {
        public string Name { get; set; } = string.Empty;
        public string Path { get; set; } = string.Empty;
        public string Content { get; set; } = string.Empty;
        public string Checksum { get; set; } = string.Empty;
    }
}