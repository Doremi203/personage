using System.Threading;
using System.Threading.Tasks;
using Dapper;
using Personage.Auth.DataAccess.Interfaces;
using Personage.Auth.DataAccess.Models;

namespace Personage.Auth.Tests.Infrastructure.Repositories;

public class TestOAuthStateRepository(IDbConnectionFactory connectionFactory)
{
    public async Task<OAuthState?> GetOAuthStateByUserEmail(string userEmail)
    {
        using var connection = await connectionFactory.CreateConnection(CancellationToken.None);
        
        return await connection.QueryFirstOrDefaultAsync<OAuthState>(
            """
            SELECT 
                state as State,
                user_email as UserEmail,
                redirect_uri as RedirectUri, 
                created_at as CreatedAt,
                expires_at as ExpiresAt
            FROM oauth_state
            WHERE user_email = @userEmail AND expires_at > NOW()
            """,
            new { userEmail });
    } 
}