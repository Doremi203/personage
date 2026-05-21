using Personage.Auth.Domain.Models.User;

namespace Personage.Auth.Domain.Interfaces;

public interface IUserProfileService
{
    Task<UserProfileModel> GetUserById(Guid id, CancellationToken ct);
}
