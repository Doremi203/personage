using System.Data;
using Microsoft.Extensions.Options;
using Npgsql;
using Personage.Auth.DataAccess.Interfaces;
using Personage.Auth.Domain.Configuration;

namespace Personage.Auth.DataAccess;

public class DbConnectionFactory(IOptions<ConnectionFactorySettings> connectionFactorySettings) : IDbConnectionFactory
{
    public async Task<IDbConnection> CreateConnection(CancellationToken token)
    {
        var connection = new NpgsqlConnection(connectionFactorySettings.Value.ConnectionString);
        await connection.OpenAsync(token);
        return connection;
    }
}