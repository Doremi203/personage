using System;
using Dapper;
using System.Threading;
using System.Threading.Tasks;
using Personage.Auth.DataAccess.Interfaces;

namespace Personage.Auth.Tests.Infrastructure;

public class TestCleaners(IDbConnectionFactory connectionFactory)
{
    public async Task DeleteUser(Guid userId) => await DeleteUsers([userId]);
    public async Task DeleteUsers(Guid[] userIds)
    {
        using var connection = await connectionFactory.CreateConnection(CancellationToken.None);

        await connection.ExecuteAsync("delete from gmail_token where user_id = any(@userIds);", new { userIds });
        await connection.ExecuteAsync("""delete from "user" where id = any(@userIds);""", new { userIds });
    }

    public async Task DeleteOAuthState(string state)
    {
        using var connection = await connectionFactory.CreateConnection(CancellationToken.None);

        await connection.ExecuteAsync("delete from oauth_state where state = @state;", new { state });
    }
}