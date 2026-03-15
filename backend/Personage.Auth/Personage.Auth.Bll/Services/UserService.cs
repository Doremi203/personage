using Personage.Auth.DataAccess.Interfaces.Repositories;
using Personage.Auth.Domain.Exceptions.Base;
using Personage.Auth.Domain.Interfaces;
using Personage.Auth.Domain.Models.User;
using Personage.Auth.Domain.Models.User.Integrations;

namespace Personage.Auth.Bll.Services;

public class UserService(
    IClaimValues claimValues,
    IUserRepository userRepository,
    ITelegramSessionRepository telegramSessionRepository,
    IGmailTokenRepository gmailTokenRepository
) : IUserService
{
    public async Task<UserInfoModel> GetUserInfo(CancellationToken ct)
    {
        var userId = claimValues.GetUserId();

        //Can be simplified to a single db call for optimisation
        var userInfo = await userRepository.GetUserById(userId, ct);
        if(userInfo is null)
            throw new NotFoundException(ErrorCode.UserNotFound, "User with specified id not found");
        
        var telegramSession = await telegramSessionRepository.GetSessionString(userId, ct);
        var gmailInfo = await gmailTokenRepository.GetTokenByUserId(userId, ct);

        return new UserInfoModel
        {
            Email = userInfo.Email,
            Name = userInfo.Name,
            GmailIntegrationModel = new GmailIntegrationModel
            {
                Enabled = gmailInfo is not null,
                Gmail = gmailInfo?.GmailEmail
            },
            TelegramIntegrationModel = new TelegramIntegrationModel
            {
                Enabled = telegramSession is not null
            }
        };
    }
}