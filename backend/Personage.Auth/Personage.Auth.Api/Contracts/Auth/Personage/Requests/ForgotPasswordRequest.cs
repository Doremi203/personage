namespace Personage.Auth.Api.Contracts.Auth.Personage.Requests;

public class ForgotPasswordRequest
{
    public string Email { get; init; } = null!;
    public string ResetUrlBase { get; init; } = null!;
}