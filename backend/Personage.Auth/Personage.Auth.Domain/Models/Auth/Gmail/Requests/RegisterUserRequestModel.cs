namespace Personage.Auth.Domain.Models.Auth.Gmail.Requests;

public class RegisterUserRequestModel
{
    public string Email { get; init; } = null!;
    public string Name { get; init; } = null!;
    public string Password { get; init; } = null!;
}