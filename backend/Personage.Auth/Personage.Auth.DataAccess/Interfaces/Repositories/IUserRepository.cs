using Personage.Auth.DataAccess.Models;
using Personage.Auth.DataAccess.Models.Requests;

namespace Personage.Auth.DataAccess.Interfaces.Repositories;

public interface IUserRepository
{
    Task<User?> GetUserByEmail(string email, CancellationToken ct);
    Task<User?> GetUserById(Guid userId, CancellationToken ct);
    Task<User> CreateShortUser(string email, CancellationToken ct);
    Task<User> CreateUser(CreateUserRequest request, CancellationToken ct);
    Task<UserWithToken[]> GetUsersProcessedBeforeMoment(DateTime processedBeforeMoment, int limit, CancellationToken ct);
    Task MarkUsersAsProcessed((Guid UserId, DateTime ProcessedAt)[] users, CancellationToken ct);
}
