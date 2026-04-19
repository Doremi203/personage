namespace Personage.Auth.Api.Contracts.Auth.OAuth.Responses;

public class StartAuthResponse
{
    public string AuthorizationUrl { get; set; } = null!;
    public string State { get; set; } = null!;
}