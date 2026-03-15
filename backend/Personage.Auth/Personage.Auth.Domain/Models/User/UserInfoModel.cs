using Personage.Auth.Domain.Models.User.Integrations;

namespace Personage.Auth.Domain.Models.User;

public class UserInfoModel
{
    public string Email { get; init; } = null!;
    public string Name { get; init; } = null!;

    public GmailIntegrationModel GmailIntegrationModel { get; init; } = null!;
    public TelegramIntegrationModel TelegramIntegrationModel { get; init; } = null!;
}