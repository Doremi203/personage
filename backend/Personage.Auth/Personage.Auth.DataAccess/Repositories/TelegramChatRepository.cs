using Dapper;
using Personage.Auth.DataAccess.Interfaces;
using Personage.Auth.DataAccess.Interfaces.Repositories;
using Personage.Auth.Domain.Models.TelegramAuth;

namespace Personage.Auth.DataAccess.Repositories;

public class TelegramChatRepository(IDbConnectionFactory connectionFactory) : ITelegramChatRepository
{
    public async Task<IReadOnlyList<TelegramChatModel>> GetUserChats(Guid userId, CancellationToken ct)
    {
        using var connection = await connectionFactory.CreateConnection(ct);

        var rows = await connection.QueryAsync<TelegramChatModel>(
            """
            --TelegramChatRepository.GetUserChats
            SELECT chat_id AS ChatId,
                   chat_name AS ChatName,
                   is_active AS IsActive
            FROM telegram_chats
            WHERE user_id = @userId
            ORDER BY chat_name;
            """,
            new { userId });

        return rows.ToList();
    }

    public async Task<IReadOnlyDictionary<Guid, IReadOnlyList<TelegramChatModel>>> GetChatsForUsers(
        IReadOnlyCollection<Guid> userIds,
        CancellationToken ct)
    {
        if (userIds.Count == 0)
            return new Dictionary<Guid, IReadOnlyList<TelegramChatModel>>();

        using var connection = await connectionFactory.CreateConnection(ct);

        var rows = await connection.QueryAsync<(Guid UserId, long ChatId, string ChatName, bool IsActive)>(
            """
            --TelegramChatRepository.GetChatsForUsers
            SELECT user_id, chat_id, chat_name, is_active
            FROM telegram_chats
            WHERE user_id = ANY(@userIds);
            """,
            new { userIds = userIds.ToArray() });

        return rows
            .GroupBy(r => r.UserId)
            .ToDictionary(
                g => g.Key,
                IReadOnlyList<TelegramChatModel> (g) => g
                    .Select(r => new TelegramChatModel
                    {
                        ChatId = r.ChatId,
                        ChatName = r.ChatName,
                        IsActive = r.IsActive
                    })
                    .ToList());
    }

    public async Task InsertNewChats(
        Guid userId,
        IReadOnlyCollection<(long ChatId, string ChatName)> chats,
        CancellationToken ct)
    {
        if (chats.Count == 0)
            return;

        using var connection = await connectionFactory.CreateConnection(ct);

        await connection.ExecuteAsync(
            """
            --TelegramChatRepository.InsertNewChats
            INSERT INTO telegram_chats (user_id, chat_id, chat_name)
            SELECT @userId, batch.chat_id, batch.chat_name
            FROM (SELECT
                unnest(@chatIds) AS chat_id,
                unnest(@chatNames) AS chat_name
            ) AS batch
            ON CONFLICT (user_id, chat_id) DO NOTHING;
            """,
            new
            {
                userId,
                chatIds = chats.Select(c => c.ChatId).ToArray(),
                chatNames = chats.Select(c => c.ChatName).ToArray()
            });
    }

    public async Task<bool> UpdateActiveStatus(Guid userId, long chatId, bool isActive, CancellationToken ct)
    {
        using var connection = await connectionFactory.CreateConnection(ct);

        var affected = await connection.ExecuteAsync(
            """
            --TelegramChatRepository.UpdateActiveStatus
            UPDATE telegram_chats
            SET is_active = @isActive
            WHERE user_id = @userId AND chat_id = @chatId;
            """,
            new { userId, chatId, isActive });

        return affected > 0;
    }

    public async Task DeleteChat(Guid userId, long chatId, CancellationToken ct)
    {
        using var connection = await connectionFactory.CreateConnection(ct);

        await connection.ExecuteAsync(
            """
            --TelegramChatRepository.DeleteChat
            DELETE FROM telegram_chats
            WHERE user_id = @userId AND chat_id = @chatId;
            """,
            new { userId, chatId });
    }
}
