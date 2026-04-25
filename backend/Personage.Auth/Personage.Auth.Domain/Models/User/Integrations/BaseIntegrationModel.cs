namespace Personage.Auth.Domain.Models.User.Integrations;

public abstract class BaseIntegrationModel
{
    public bool Enabled { get; init; }
}