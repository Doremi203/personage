namespace Personage.Auth.Domain.Configuration;

public class OAuthSettings
{
    public string ClientId { get; set; } = null!;
    public string ClientSecret { get; set; } = null!;
    public string[] Scopes { get; set; } = []; // gmail scopes
    public string[] GoogleCalendarScopes { get; set; } = [];
}