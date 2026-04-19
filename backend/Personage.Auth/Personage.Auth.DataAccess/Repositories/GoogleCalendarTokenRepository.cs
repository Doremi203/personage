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
        
        //TODO: gmail not saved?
        
        await connection.ExecuteAsync(
            """
            --GmailTokenRepository.SaveToken
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
}