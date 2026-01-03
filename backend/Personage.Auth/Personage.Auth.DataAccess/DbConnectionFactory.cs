using System.Data;
using Microsoft.Extensions.Configuration;
using Npgsql;
using Personage.Auth.DataAccess.Interfaces;

namespace Personage.Auth.DataAccess;

public class DbConnectionFactory(IConfiguration configuration) : IDbConnectionFactory
{
    private readonly string _connectionString =
        configuration.GetConnectionString("DefaultConnection")
        ?? throw new ArgumentNullException("Connection string not found");


    public async Task<IDbConnection> CreateConnection(CancellationToken token)
    {
        var connection = new NpgsqlConnection(_connectionString);
        await connection.OpenAsync(token);
        return connection;
    }
}