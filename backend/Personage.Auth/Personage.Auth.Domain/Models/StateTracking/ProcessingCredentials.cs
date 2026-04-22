using Personage.Auth.Domain.Models.Common;

namespace Personage.Auth.Domain.Models.StateTracking;

public abstract class ProcessingCredentialsBase;

public sealed class GmailProcessingCredentials : ProcessingCredentialsBase
{
    public OAuthTokenModel Tokens { get; init; } = null!;
}

public sealed class GoogleCalendarProcessingCredentials : ProcessingCredentialsBase
{
    public OAuthTokenModel Tokens { get; init; } = null!;
}

public sealed class TelegramProcessingCredentials : ProcessingCredentialsBase
{
    public string SessionString { get; init; } = null!;
}