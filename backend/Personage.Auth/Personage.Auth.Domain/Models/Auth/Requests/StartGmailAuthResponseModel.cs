namespace Personage.Auth.Domain.Models.Auth.Requests;

public class StartGmailAuthResponseModel
{
    public string Uri { get; init; } = null!;
    public string State { get; init; } = null!;
}