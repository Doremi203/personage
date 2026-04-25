namespace Personage.Auth.DataAccess.Interfaces.Repositories;

public interface ITelegramSessionRepository
{
    Task<Guid> StoreSession(Guid userId, string sessionString, CancellationToken ct);
    Task<string?> GetSessionString(Guid userId, CancellationToken ct);
    Task MarkUsersAsProcessed((Guid UserId, DateTime ProcessedAt)[] users, CancellationToken ct);
    Task RemoveSession(Guid userId, CancellationToken ct);
}