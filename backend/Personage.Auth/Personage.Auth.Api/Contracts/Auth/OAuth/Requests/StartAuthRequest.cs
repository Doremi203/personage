namespace Personage.Auth.Api.Contracts.Auth.OAuth.Requests;

public class StartAuthRequest
{
    public string UserEmail { get; set; } = null!;
    public string RedirectUri { get; set; } = null!;
}