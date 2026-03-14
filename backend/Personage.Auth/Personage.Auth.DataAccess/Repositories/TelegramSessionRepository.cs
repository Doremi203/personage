using Dapper;
using Personage.Auth.DataAccess.Interfaces;
using Personage.Auth.DataAccess.Interfaces.Repositories;

namespace Personage.Auth.DataAccess.Repositories;

public class TelegramSessionRepository(IDbConnectionFactory connectionFactory) : ITelegramSessionRepository
{
    public async Task<Guid> StoreSession(Guid userId, string sessionString, CancellationToken ct)
    {
        using var connection = await connectionFactory.CreateConnection(ct);
        
        return await connection.QueryFirstAsync<Guid>(
            """
            INSERT INTO telegram_session (user_id, session) 
            VALUES (@userId, @sessionString)
            ON CONFLICT (user_id) 
            DO UPDATE SET 
                session = EXCLUDED.session,
                created_at = NOW()
            RETURNING id;
            """,
            new { userId, sessionString});
    }

    public async Task<string?> GetSessionString(Guid userId, CancellationToken ct)
    {
        using var connection = await connectionFactory.CreateConnection(ct);
        
        return await connection.QueryFirstOrDefaultAsync<string>(
            """
            SELECT session 
            from telegram_session
            where user_id = @userId;
            """,
            new { userId });
    }
}