using Personage.Auth.DataAccess.Interfaces.Repositories;
using Personage.Auth.Domain.Exceptions.Base;
using Personage.Auth.Domain.Interfaces;
using Personage.Auth.Domain.Models.User;

namespace Personage.Auth.Bll.Services;

public class UserProfileService(IUserRepository userRepository) : IUserProfileService
{
    public async Task<UserProfileModel> GetUserById(Guid id, CancellationToken ct)
    {
        var user = await userRepository.GetUserById(id, ct);
        if (user is null)
            throw new NotFoundException(ErrorCode.UserNotFound, "User with specified id not found");

        return new UserProfileModel
        {
            Id = user.Id,
            Email = user.Email,
            Name = user.Name,
            CreatedAt = user.CreatedAt,
        };
    }
}
