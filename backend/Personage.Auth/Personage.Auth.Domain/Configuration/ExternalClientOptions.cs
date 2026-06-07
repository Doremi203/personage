namespace Personage.Auth.Domain.Configuration;

public class ExternalClientOptions
{
    public int MaxRefreshRetryAttempts { get; set; } = 3;
}