namespace Personage.Auth.Domain.Configuration;

public class PostboxSettings
{
    public string FromEmail { get; init; } = null!;
    public string FromName { get; init; } = "Personage Auth";
    public string SecretId { get; init; } = null!;
}