using Dapper;
using Personage.Auth.DataAccess.Interfaces;
using Personage.Auth.DataAccess.Interfaces.Repositories;
using Personage.Auth.DataAccess.Models;

namespace Personage.Auth.DataAccess.Repositories;

public class GoogleCalendarTokenRepository(IDbConnectionFactory connectionFactory) : IGoogleCalendarTokenRepository
{
    public async Task SaveToken(OAuthToken token, CancellationToken ct)
    {
        using var connection = await connectionFactory.CreateConnection(ct);

        await connection.ExecuteAsync(
            """
            --GoogleCalendarTokenRepository.SaveToken
            INSERT INTO calendar_token (
                user_id,
                access_token,
                refresh_token,
                expires_at,
                gmail_email,
                status
            )
            VALUES (@UserId, @AccessToken, @RefreshToken, @ExpiresAt, @GmailEmail, @Status)
            ON CONFLICT (user_id) DO UPDATE SET
                access_token = EXCLUDED.access_token,
                refresh_token = EXCLUDED.refresh_token,
                expires_at = EXCLUDED.expires_at,
                gmail_email = EXCLUDED.gmail_email;
            """,
            token);
    }

    public async Task MarkUsersAsProcessed((Guid UserId, DateTime ProcessedAt)[] users, CancellationToken ct)
    {
        using var connection = await connectionFactory.CreateConnection(ct);

        await connection.ExecuteAsync(
            """
            --GoogleCalendarTokenRepository.MarkUsersAsProcessed
            UPDATE calendar_token ct
            SET last_processed_at = batch.processed_at
            FROM (select
                unnest(@userIds) AS user_id,
                unnest(@processedAtMoments) AS processed_at
            ) AS batch
            WHERE ct.user_id = batch.user_id;
            """,
            new
            {
                userIds = users.Select(u => u.UserId).ToArray(),
                processedAtMoments = users.Select(u => u.ProcessedAt).ToArray()
            });
    }

    public async Task<OAuthTokenWithId?> GetTokenByUserId(Guid userId, CancellationToken ct)
    {
        using var connection = await connectionFactory.CreateConnection(ct);

        return await connection.QueryFirstOrDefaultAsync<OAuthTokenWithId>(
            """
            --GoogleCalendarTokenRepository.GetTokenByUserId
            SELECT
                ct.id as Id,
                ct.user_id as UserId,
                ct.access_token as AccessToken, 
                ct.refresh_token as RefreshToken,
                ct.expires_at as ExpiresAt,
                ct.gmail_email as GmailEmail,
                ct.created_at as CreatedAt
            FROM calendar_token ct
            WHERE ct.user_id = @userId;
            """,
            new { userId });
    }

    public async Task RemoveToken(Guid userId, CancellationToken ct)
    {
        using var connection = await connectionFactory.CreateConnection(ct);

        await connection.ExecuteAsync(
            """
            --GoogleCalendarTokenRepository.RemoveToken
            DELETE FROM calendar_token
            WHERE user_id = @userId;
            """,
            new
            {
                userId
            });
    }

    public async Task RemoveTokens(Guid[] tokenIds, CancellationToken ct)
    {
        using var connection = await connectionFactory.CreateConnection(ct);

        await connection.ExecuteAsync(
            """
            --GoogleCalendarTokenRepository.RemoveTokens
            DELETE FROM calendar_token
            WHERE id = any(@tokenIds);
            """,
            new
            {
                tokenIds
            });
    }
    
    public async Task UpdateToken(Guid tokenId, string accessToken, string refreshToken, DateTime expiresAt, CancellationToken ct)
    {
        using var connection = await connectionFactory.CreateConnection(ct);

        await connection.ExecuteAsync(
            """
            --GoogleCalendarTokenRepository.UpdateToken
            UPDATE calendar_token
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

    public async Task<(Guid TokenId, int FailedAttempts)[]> UpdateRefreshInfo((Guid tokenId, bool RefreshSuccess)[] refreshes, CancellationToken ct)
    {
        using var connection = await connectionFactory.CreateConnection(ct);

        var result = await connection.QueryAsync<(Guid TokenId, int FailedAttempts)>(
            """
            --GoogleCalendarTokenRepository.UpdateRefreshInfo
            UPDATE calendar_token AS ct
            SET
                failed_refreshes = CASE 
                    WHEN refresh_data.refresh_success THEN 0 
                    ELSE ct.failed_refreshes + 1 
                END,
                last_refresh_attempt = NOW()
            FROM UNNEST(@Refreshes) AS refresh_data(token_id UUID, refresh_success BOOLEAN)
            WHERE ct.id = refresh_data.token_id
            RETURNING ct.id AS token_id, ct.failed_refreshes;
            """,
            new
            {
                Refreshes = refreshes.Select(r => new { r.tokenId, r.RefreshSuccess }).ToArray()
            });
        
        return result.ToArray();
    }
}