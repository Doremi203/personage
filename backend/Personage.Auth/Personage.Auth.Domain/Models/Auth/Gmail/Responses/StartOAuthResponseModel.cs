namespace Personage.Auth.Domain.Models.Auth.Gmail.Responses;

public class StartOAuthResponseModel
{
    public string Uri { get; init; } = null!;
    public string State { get; init; } = null!;
}