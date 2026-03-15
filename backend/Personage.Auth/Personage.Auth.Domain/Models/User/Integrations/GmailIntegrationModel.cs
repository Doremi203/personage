namespace Personage.Auth.Domain.Models.User.Integrations;

public class GmailIntegrationModel
{
    public bool Enabled { get; init; }
    public string? Gmail { get; init; }
}