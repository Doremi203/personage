namespace Personage.Auth.Api.Contracts.Auth.Gmail.Responses;

public class StartAuthResponse
{
    public string AuthorizationUrl { get; set; } = null!;
    public string State { get; set; } = null!;
}