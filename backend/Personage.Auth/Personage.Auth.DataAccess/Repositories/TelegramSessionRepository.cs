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
            new { userId, sessionString });
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

    public async Task MarkUsersAsProcessed((Guid UserId, DateTime ProcessedAt)[] users, CancellationToken ct)
    {
        using var connection = await connectionFactory.CreateConnection(ct);

        await connection.ExecuteAsync(
            """
            --TelegramSessionRepository.MarkUsersAsProcessed
            UPDATE telegram_session ts
            SET last_processed_at = batch.processed_at
            FROM (select
                unnest(@userIds) AS user_id,
                unnest(@processedAtMoments) AS processed_at
            ) AS batch
            WHERE ts.user_id = batch.user_id;
            """,
            new
            {
                userIds = users.Select(u => u.UserId).ToArray(),
                processedAtMoments = users.Select(u => u.ProcessedAt).ToArray()
            });
    }

    public async Task RemoveSession(Guid userId, CancellationToken ct)
    {
        using var connection = await connectionFactory.CreateConnection(ct);

        await connection.ExecuteAsync(
            """
            --TelegramSessionRepository.RemoveSession
            DELETE FROM telegram_session
            WHERE user_id = @userId;
            """,
            new
            {
                userId
            });
    }
}