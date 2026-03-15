namespace Personage.Auth.Api.Contracts.User.Integrations;

public class GmailIntegration
{
    public bool Enabled { get; init; }
    public string? Gmail { get; init; }
}