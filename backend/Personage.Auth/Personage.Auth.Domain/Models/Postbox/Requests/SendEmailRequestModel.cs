namespace Personage.Auth.Domain.Models.Postbox.Requests;

public class SendEmailRequestModel
{
    public string To { get; init; } = null!;
    public string Subject { get; init; } = null!;
    public string HtmlBody { get; init; } = null!;
}