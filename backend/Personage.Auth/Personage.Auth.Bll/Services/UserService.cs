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
    IGmailTokenRepository gmailTokenRepository,
    IGoogleCalendarTokenRepository googleCalendarTokenRepository
) : IUserService
{
    public async Task<UserInfoModel> GetUserInfo(CancellationToken ct)
    {
        var userId = claimValues.GetUserId();

        //TODO: Can be simplified to a single db call for optimisation
        var userInfo = await userRepository.GetUserById(userId, ct);
        if(userInfo is null)
            throw new NotFoundException(ErrorCode.UserNotFound, "User with specified id not found");
        
        var telegramSession = await telegramSessionRepository.GetSessionString(userId, ct);
        var gmailInfo = await gmailTokenRepository.GetTokenByUserId(userId, ct);
        var googleCalendarInfo = await googleCalendarTokenRepository.GetTokenByUserId(userId, ct);

        return new UserInfoModel
        {
            Email = userInfo.Email,
            Name = userInfo.Name,
            GmailIntegration = new GmailIntegrationModel
            {
                Enabled = gmailInfo is not null,
                Gmail = gmailInfo?.GmailEmail
            },
            TelegramIntegration = new TelegramIntegrationModel
            {
                Enabled = telegramSession is not null
            },
            GoogleCalendarIntegration = new GoogleCalendarIntegrationModel
            {
                Enabled = googleCalendarInfo is not null,
                Gmail = googleCalendarInfo?.GmailEmail
            }
        };
    }
}