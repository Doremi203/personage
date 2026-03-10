namespace Personage.Auth.DataAccess.Models;

public class User
{
    public Guid Id { get; init; }
    public string Email { get; init; } = null!;
    public string Name { get; init; } = null!;
    public DateTime CreatedAt { get; init; } = DateTime.UtcNow;
    public string? PasswordHash { get; init; }
}