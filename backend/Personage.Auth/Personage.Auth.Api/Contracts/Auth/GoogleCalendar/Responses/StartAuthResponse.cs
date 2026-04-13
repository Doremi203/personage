namespace Personage.Auth.Api.Contracts.Auth.GoogleCalendar.Responses;

public class StartAuthResponse
{
    public string AuthorizationUrl { get; set; } = null!;
    public string State { get; set; } = null!;
}