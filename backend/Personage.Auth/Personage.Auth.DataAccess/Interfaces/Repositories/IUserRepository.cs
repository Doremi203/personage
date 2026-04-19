using Personage.Auth.DataAccess.Models;
using Personage.Auth.DataAccess.Models.Requests;

namespace Personage.Auth.DataAccess.Interfaces.Repositories;

public interface IUserRepository
{
    Task<User?> GetUserByEmail(string email, CancellationToken ct);
    Task<User?> GetUserById(Guid userId, CancellationToken ct);
    Task<User> CreateShortUser(string email, CancellationToken ct);
    Task<User> CreateUser(CreateUserRequest request, CancellationToken ct);
    Task<UserWithToken[]> GetUsersGmailProcessedBeforeMoment(DateTime processedBeforeMoment, int limit, CancellationToken ct);
    Task<UserWithTelegramSession[]> GetUsersTelegramProcessedBeforeMoment(DateTime processedBeforeMoment, int limit, CancellationToken ct);
    Task<UserWithToken[]> GetUsersGoogleCalendarProcessedBeforeMoment(DateTime processedBeforeMoment, int limit, CancellationToken ct);
    Task UpdatePassword(Guid userId, string passwordHash, CancellationToken ct);
}
