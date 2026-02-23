namespace Personage.Auth.Domain.Models.Common;

public class PersonageTokenModel
{
    public string AccessToken { get; init; } = null!;
    public string RefreshToken { get; init; } = null!;
}