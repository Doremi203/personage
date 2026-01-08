namespace Personage.Auth.DataAccess.Models;

public class User
{
    public string Id { get; init; } = null!;
    public string Email { get; init; } = null!;
    public DateTime CreatedAt { get; init; } = DateTime.UtcNow;
}