namespace Personage.Auth.Api.Contracts.Auth.Personage.Requests;

public class LoginWithPasswordRequest
{
    public string Email { get; init; } = null!;
    public string Password { get; init; } = null!;
}