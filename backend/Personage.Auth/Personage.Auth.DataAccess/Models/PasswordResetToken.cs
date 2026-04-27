namespace Personage.Auth.DataAccess.Models;

public class PasswordResetToken
{
    public Guid Id { get; init; }
    public Guid UserId { get; init; }
    public string Token { get; init; } = null!;
    public DateTime ExpiresAt { get; init; }
    public DateTime CreatedAt { get; init; }
    public DateTime? UsedAt { get; init; }
}