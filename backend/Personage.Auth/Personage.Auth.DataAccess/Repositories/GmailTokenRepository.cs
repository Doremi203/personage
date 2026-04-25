using Dapper;
using Personage.Auth.DataAccess.Interfaces;
using Personage.Auth.DataAccess.Interfaces.Repositories;
using Personage.Auth.DataAccess.Models;

namespace Personage.Auth.DataAccess.Repositories;

public class GmailTokenRepository(IDbConnectionFactory connectionFactory) : IGmailTokenRepository
{
    public async Task<OAuthTokenWithId?> GetTokenByUserEmail(string userEmail, CancellationToken ct)
    {
        using var connection = await connectionFactory.CreateConnection(ct);
        
        return await connection.QueryFirstOrDefaultAsync<OAuthTokenWithId>(
            """
            --GmailTokenRepository.GetTokenByUserEmail
            SELECT
                gt.id as Id,
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

    public async Task<OAuthTokenWithId?> GetTokenByUserId(Guid userId, CancellationToken ct)
    {
        using var connection = await connectionFactory.CreateConnection(ct);
        
        return await connection.QueryFirstOrDefaultAsync<OAuthTokenWithId>(
            """
            --GmailTokenRepository.GetTokenByUserId
            SELECT
                gt.id as Id,
                gt.user_id as UserId,
                gt.access_token as AccessToken, 
                gt.refresh_token as RefreshToken,
                gt.expires_at as ExpiresAt,
                gt.gmail_email as GmailEmail,
                gt.created_at as CreatedAt
            FROM gmail_token gt
            WHERE gt.user_id = @userId;
            """,
            new { userId });
    }

    public async Task SaveToken(OAuthToken token, CancellationToken ct)
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

    public async Task UpdateToken(Guid tokenId, string accessToken, string refreshToken, DateTime expiresAt, CancellationToken ct)
    {
        using var connection = await connectionFactory.CreateConnection(ct);
        
        await connection.ExecuteAsync(
            """
            --GmailTokenRepository.UpdateToken
            UPDATE gmail_token
            SET
                access_token = @accessToken,
                refresh_token = @refreshToken,
                expires_at = @expiresAt
            WHERE
                id = @tokenId;
            """,
            new
            {
                accessToken,
                refreshToken,
                expiresAt,
                tokenId
            });
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
    
    public async Task MarkUsersAsProcessed((Guid UserId, DateTime ProcessedAt)[] users, CancellationToken ct)
    {
        using var connection = await connectionFactory.CreateConnection(ct);
        
        await connection.ExecuteAsync(
            """
            --GmailTokenRepository.MarkUsersAsProcessed
            UPDATE gmail_token gt
            SET last_processed_at = batch.processed_at
            FROM (select
                unnest(@userIds) AS user_id,
                unnest(@processedAtMoments) AS processed_at
            ) AS batch
            WHERE gt.user_id = batch.user_id;
            """,
            new
            {
                userIds = users.Select(u => u.UserId).ToArray(),
                processedAtMoments = users.Select(u => u.ProcessedAt).ToArray()
            });
    }

    public async Task RemoveToken(Guid userId, CancellationToken ct)
    {
        using var connection = await connectionFactory.CreateConnection(ct);
        
        await connection.ExecuteAsync(
            """
            --GmailTokenRepository.RemoveToken
            DELETE FROM gmail_token
            WHERE user_id = @userId;
            """,
            new
            {
                userId
            });
    }
}