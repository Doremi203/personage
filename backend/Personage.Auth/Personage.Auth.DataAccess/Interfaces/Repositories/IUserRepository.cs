using Personage.Auth.DataAccess.Models;

namespace Personage.Auth.DataAccess.Interfaces.Repositories;

public interface IUserRepository
{
    Task<User?> GetUserByEmail(string email, CancellationToken ct);
    Task<User> CreateUser(string email, CancellationToken ct);
}
