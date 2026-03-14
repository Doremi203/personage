namespace Personage.Auth.DataAccess.Interfaces.Repositories;

public interface ITelegramSessionRepository
{
    public Task<Guid> StoreSession(Guid userId, string sessionString, CancellationToken ct);
    public Task<string?> GetSessionString(Guid userId, CancellationToken ct);
}