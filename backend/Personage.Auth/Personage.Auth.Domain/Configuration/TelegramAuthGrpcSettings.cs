namespace Personage.Auth.Domain.Configuration;

public class TelegramAuthGrpcSettings
{
    public string Url { get; init; } = null!;
    public int TimeoutSeconds { get; init; } = 60;
}
