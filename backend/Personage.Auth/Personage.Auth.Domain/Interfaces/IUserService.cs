using Personage.Auth.Domain.Models.User;

namespace Personage.Auth.Domain.Interfaces;

public interface IUserService
{
    Task<UserInfoModel> GetUserInfo(CancellationToken ct);
    Task UpdateUserInfo(string name, CancellationToken ct);
}