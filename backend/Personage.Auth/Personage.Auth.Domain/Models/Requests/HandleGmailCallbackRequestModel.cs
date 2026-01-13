namespace Personage.Auth.Domain.Models.Requests;

public class HandleGmailCallbackRequestModel
{
    public string UserEmail { get; init; } = null!;
    public string Code { get; init; } = null!;
    public string State { get; init; } = null!;
    public string RedirectUri { get; init; } = null!;
}