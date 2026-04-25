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
}