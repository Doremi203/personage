namespace Personage.Auth.Api.Contracts.Auth.Personage.Requests;

public class ResetPasswordRequest
{
    public string Token { get; init; } = null!;
    public string NewPassword { get; init; } = null!;
}