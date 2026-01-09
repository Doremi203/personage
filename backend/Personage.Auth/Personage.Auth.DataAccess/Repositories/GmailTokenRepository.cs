using Dapper;
using Personage.Auth.DataAccess.Interfaces;
using Personage.Auth.DataAccess.Interfaces.Repositories;
using Personage.Auth.DataAccess.Models;

namespace Personage.Auth.DataAccess.Repositories;

public class GmailTokenRepository(IDbConnectionFactory connectionFactory) : IGmailTokenRepository
{
    public async Task<GmailToken?> GetToken(Guid userId, CancellationToken ct)
    {
        using var connection = await connectionFactory.CreateConnection(ct);
        
        return await connection.QueryFirstOrDefaultAsync<GmailToken>(
            """
            SELECT 
                user_id as UserId,
                access_token as AccessToken, 
                refresh_token as RefreshToken,
                expires_at as ExpiresAt,
                gmail_email as GmailEmail,
                created_at as CreatedAt
            FROM gmail_token
            WHERE user_id = @userId;
            """,
            new { userId });
    }

    public async Task SaveToken(GmailToken token, CancellationToken ct)
    {
        using var connection = await connectionFactory.CreateConnection(ct);
        
        await connection.ExecuteAsync(
            """
            INSERT INTO gmail_token (user_id, access_token, refresh_token, expires_at, gmail_email)
            VALUES (@UserId, @AccessToken, @RefreshToken, @ExpiresAt, @GmailEmail)
            ON CONFLICT (user_id) DO UPDATE SET
                access_token = EXCLUDED.access_token,
                refresh_token = EXCLUDED.refresh_token,
                expires_at = EXCLUDED.expires_at,
                gmail_email = EXCLUDED.gmail_email;
            """,
            token);
    }
}