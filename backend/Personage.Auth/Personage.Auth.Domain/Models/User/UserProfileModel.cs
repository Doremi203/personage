namespace Personage.Auth.Domain.Models.User;

public class UserProfileModel
{
    public Guid Id { get; init; }
    public string Email { get; init; } = null!;
    public string Name { get; init; } = null!;
    public DateTime CreatedAt { get; init; }
    public IReadOnlyList<string> ConnectedEmails { get; init; } = Array.Empty<string>();
}
