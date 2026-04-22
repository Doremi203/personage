namespace Personage.Auth.Api.Contracts.Auth.OAuth.Requests;

public class AuthCallbackRequest
{
    public string UserEmail { get; init; } = string.Empty;
    public string Code { get; init; } = string.Empty;
    public string State { get; init; } = string.Empty;
    public string RedirectUri { get; init; } = string.Empty;
}