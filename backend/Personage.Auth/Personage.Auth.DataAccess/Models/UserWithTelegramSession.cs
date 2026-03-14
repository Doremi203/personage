namespace Personage.Auth.DataAccess.Models;

public class UserWithTelegramSession
{
    public Guid UserId { get; init; }
    public DateTime? LastProcessedAt { get; init; }
    public string SessionString { get; set; } = null!;
}