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
        
        /*
         *
         * id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
           user_id       UUID NOT NULL REFERENCES "user" (id) ON DELETE CASCADE,
           access_token  TEXT NOT NULL,
           refresh_token TEXT NOT NULL,
           expires_at    TIMESTAMPTZ NOT NULL,
           created_at    TIMESTAMPTZ DEFAULT NOW(),
           updated_at    TIMESTAMPTZ DEFAULT NOW(),
           last_processed_at TIMESTAMPTZ,
           status        SMALLINT NOT NULL,
           UNIQUE (user_id)
         */
        
        
        await connection.ExecuteAsync(
            """
            --GmailTokenRepository.SaveToken
            INSERT INTO calendar_tokens (
                user_id,
                access_token,
                refresh_token,
                expires_at,
                gmail_email
            )
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