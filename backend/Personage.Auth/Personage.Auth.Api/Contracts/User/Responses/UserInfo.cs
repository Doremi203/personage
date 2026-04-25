using Personage.Auth.Api.Contracts.User.Integrations;

namespace Personage.Auth.Api.Contracts.User.Responses;

public class UserInfo
{
    public string Email { get; init; } = null!;
    public string Name { get; init; } = null!;

    public GmailIntegration GmailIntegration { get; init; } = null!;
    public TelegramIntegration TelegramIntegration { get; init; } = null!;
    public GoogleCalendarIntegration GoogleCalendarIntegration { get; init; } = null!;
}