namespace Personage.Auth.DataAccess.Models.Requests;

public class CreateUserRequest
{
    public string Email { get; set; } = null!;
    public string PasswordHash { get; set; } = null!;
    public string Name { get; set; } = null!;
}