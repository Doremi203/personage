namespace Personage.Auth.Api.Contracts.Auth.GoogleCalendar.Requests;

public class StartAuthRequest
{
    public string RedirectUri { get; set; } = null!;
}