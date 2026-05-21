namespace Personage.Auth.Domain.Interfaces;

public interface ITelegramChatsGrpcClient
{
    Task<IReadOnlyList<(long Id, string Name)>> GetUserChats(
        string sessionString,
        CancellationToken ct);
}
