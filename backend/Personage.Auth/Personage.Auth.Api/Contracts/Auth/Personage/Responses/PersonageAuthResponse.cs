namespace Personage.Auth.Api.Contracts.Auth.Personage.Responses;

public class PersonageAuthResponse
{
    public string AccessToken { get; init; } = null!;
    public string? RefreshToken { get; init; }
}