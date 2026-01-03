using Personage.Auth.DataAccess.Models;

namespace Personage.Auth.DataAccess.Interfaces.Repositories;

public interface IUserRepository
{
    Task<User?> GetUserAsync(string userId, CancellationToken ct);
    Task<User> CreateOrUpdateUserAsync(string userId, string email, CancellationToken ct);
}
