using Personage.Auth.DataAccess.Interfaces.Repositories;
using Personage.Auth.Domain.Exceptions.Base;
using Personage.Auth.Domain.Interfaces;
using Personage.Auth.Domain.Models.User;

namespace Personage.Auth.Bll.Services;

public class UserProfileService(
    IUserRepository userRepository,
    IGmailTokenRepository gmailTokenRepository
) : IUserProfileService
{
    public async Task<UserProfileModel> GetUserById(Guid id, CancellationToken ct)
    {
        var user = await userRepository.GetUserById(id, ct);
        if (user is null)
            throw new NotFoundException(ErrorCode.UserNotFound, "User with specified id not found");

        var gmailToken = await gmailTokenRepository.GetTokenByUserId(id, ct);
        var connectedEmails = new List<string>();
        if (!string.IsNullOrWhiteSpace(gmailToken?.GmailEmail) &&
            !string.Equals(gmailToken.GmailEmail, user.Email, StringComparison.OrdinalIgnoreCase))
        {
            connectedEmails.Add(gmailToken.GmailEmail);
        }

        return new UserProfileModel
        {
            Id = user.Id,
            Email = user.Email,
            Name = user.Name,
            CreatedAt = user.CreatedAt,
            ConnectedEmails = connectedEmails,
        };
    }
}
