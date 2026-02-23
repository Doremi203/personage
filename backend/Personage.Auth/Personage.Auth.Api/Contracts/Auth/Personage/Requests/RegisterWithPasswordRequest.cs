namespace Personage.Auth.Api.Contracts.Auth.Personage.Requests;

public class RegisterWithPasswordRequest
{
    public string Email { get; init; } = null!;
    public string Password { get; init; } = null!;
    public string Name { get; init; } = null!;
}