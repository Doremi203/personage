namespace Personage.Auth.Contracts.Auth.Gmail.Requests;

public class StartAuthRequest
{
    public string UserEmail { get; set; } = null!;
    public string RedirectUri { get; set; } = null!;
}