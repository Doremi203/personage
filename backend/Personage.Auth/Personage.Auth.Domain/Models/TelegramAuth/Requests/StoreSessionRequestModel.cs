namespace Personage.Auth.Domain.Models.TelegramAuth.Requests;

public class StoreSessionRequestModel
{
    public Guid UserId { get; init; }
    public string SessionString { get; init; } = null!;
}