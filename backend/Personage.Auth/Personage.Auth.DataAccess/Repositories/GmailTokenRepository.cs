using Dapper;
using Personage.Auth.DataAccess.Interfaces;
using Personage.Auth.DataAccess.Interfaces.Repositories;
using Personage.Auth.DataAccess.Models;

namespace Personage.Auth.DataAccess.Repositories;

public class GmailTokenRepository(IDbConnectionFactory connectionFactory) : IGmailTokenRepository
{
    public async Task<GmailToken?> GetTokenByUserEmail(string userEmail, CancellationToken ct)
    {
        using var connection = await connectionFactory.CreateConnection(ct);
        
        return await connection.QueryFirstOrDefaultAsync<GmailToken>(
            """
            --GmailTokenRepository.GetTokenByUserEmail
            SELECT 
                gt.user_id as UserId,
                gt.access_token as AccessToken, 
                gt.refresh_token as RefreshToken,
                gt.expires_at as ExpiresAt,
                gt.gmail_email as GmailEmail,
                gt.created_at as CreatedAt
            FROM gmail_token gt
            JOIN "user" u
            on u.id = gt.user_id
            WHERE u.email = @userEmail;
            """,
            new { userEmail });
    }

    public async Task SaveToken(GmailToken token, CancellationToken ct)
    {
        using var connection = await connectionFactory.CreateConnection(ct);
        
        await connection.ExecuteAsync(
            """
            --GmailTokenRepository.SaveToken
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

    public async Task<Guid[]> GetUsersWithoutToken(Guid[] userIds, CancellationToken ct)
    {
        using var connection = await connectionFactory.CreateConnection(ct);
        
        var usersWithTokens = await connection.QueryAsync<Guid>(
            """
            --GmailTokenRepository.GetUsersWithoutToken
            SELECT 
                gt.user_id as UserId
            FROM gmail_token gt
            WHERE gt.user_id = any(@userIds);
            """,
            new { userIds });
        
        return userIds
            .Except(usersWithTokens)
            .ToArray();
    }
}