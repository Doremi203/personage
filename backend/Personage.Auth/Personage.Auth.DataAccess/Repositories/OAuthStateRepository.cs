using Dapper;
using Personage.Auth.DataAccess.Interfaces;
using Personage.Auth.DataAccess.Interfaces.Repositories;
using Personage.Auth.DataAccess.Models;

namespace Personage.Auth.DataAccess.Repositories;

public class OAuthStateRepository(IDbConnectionFactory connectionFactory) : IOAuthStateRepository
{
    public async Task<OAuthState?> GetState(string state, CancellationToken ct)
    {
        using var connection = await connectionFactory.CreateConnection(ct);
        
        return await connection.QueryFirstOrDefaultAsync<OAuthState>(
            """
            SELECT 
                state as State,
                user_email as UserEmail,
                redirect_uri as RedirectUri, 
                created_at as CreatedAt,
                expires_at as ExpiresAt
            FROM oauth_state
            WHERE state = @state AND expires_at > NOW()
            """,
            new { state });
    }

    public async Task SaveState(OAuthState state, CancellationToken ct)
    {
        using var connection = await connectionFactory.CreateConnection(ct);
        
        await connection.ExecuteAsync(
            """
            INSERT INTO oauth_state (state, user_email, redirect_uri, expires_at)
            VALUES (@State, @UserEmail, @RedirectUri, @ExpiresAt)
            ON CONFLICT (state) DO UPDATE SET
                user_email = EXCLUDED.user_email,
                redirect_uri = EXCLUDED.redirect_uri,
                expires_at = EXCLUDED.expires_at
            """,
            state);
    }

    public async Task DeleteState(string state, CancellationToken ct)
    {
        using var connection = await connectionFactory.CreateConnection(ct);
        
        await connection.ExecuteAsync(
            "DELETE FROM oauth_state WHERE state = @state",
            new { state });
    }
}