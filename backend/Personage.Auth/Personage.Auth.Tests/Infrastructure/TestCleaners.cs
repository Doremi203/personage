using System;
using Dapper;
using System.Threading;
using System.Threading.Tasks;
using Personage.Auth.DataAccess.Interfaces;

namespace Personage.Auth.Tests.Infrastructure;

public class TestCleaners(IDbConnectionFactory connectionFactory)
{
    public async Task DeleteUser(Guid userId)
    {
        using var connection = await connectionFactory.CreateConnection(CancellationToken.None);

        await connection.ExecuteAsync("""delete from "user" where id = @userId;""", new { userId });
        await connection.ExecuteAsync("delete from gmail_token where user_id = @userId", new { userId });
    }
}