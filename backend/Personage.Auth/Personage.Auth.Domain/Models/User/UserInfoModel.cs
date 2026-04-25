using Personage.Auth.Domain.Models.User.Integrations;

namespace Personage.Auth.Domain.Models.User;

public class UserInfoModel
{
    public string Email { get; init; } = null!;
    public string Name { get; init; } = null!;

    public GmailIntegrationModel GmailIntegration { get; init; } = null!;
    public TelegramIntegrationModel TelegramIntegration { get; init; } = null!;
    public GoogleCalendarIntegrationModel GoogleCalendarIntegration { get; init; } = null!;
}