using Personage.Auth.DataAccess.Models;

namespace Personage.Auth.DataAccess.Interfaces.Repositories;

public interface IUserTokenRepository
{
    Task<UserToken?> GetUserToken(string userId, ServiceType serviceType);
    Task<UserToken?> GetUserTokenByState(string state);
    Task SaveUserToken(UserToken token);
    Task UpdateUserToken(UserToken token);
    Task DeleteUserToken(string userId, ServiceType serviceType);
    Task CleanupExpiredStates();
}