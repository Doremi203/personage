using Personage.Auth.Domain.Models.TelegramAuth;

namespace Personage.Auth.Domain.Interfaces;

public interface ITelegramChatsService
{
    Task<IReadOnlyList<TelegramChatModel>> GetUserChats(Guid userId, CancellationToken ct);

    Task UpdateActiveStatus(Guid userId, long chatId, bool isActive, CancellationToken ct);
}
