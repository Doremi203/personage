using Personage.Auth.Domain.Models.TelegramAuth;

namespace Personage.Auth.DataAccess.Interfaces.Repositories;

public interface ITelegramChatRepository
{
    Task<IReadOnlyList<TelegramChatModel>> GetUserChats(Guid userId, CancellationToken ct);

    Task<IReadOnlyDictionary<Guid, IReadOnlyList<TelegramChatModel>>> GetChatsForUsers(
        IReadOnlyCollection<Guid> userIds,
        CancellationToken ct);

    Task InsertNewChats(
        Guid userId,
        IReadOnlyCollection<(long ChatId, string ChatName)> chats,
        CancellationToken ct);

    Task<bool> UpdateActiveStatus(Guid userId, long chatId, bool isActive, CancellationToken ct);

    Task DeleteChat(Guid userId, long chatId, CancellationToken ct);
}
