namespace Personage.Auth.DataAccess.Models.Requests;

public class CreateRefreshTokenRequest
{
    public string Token { get; set; } = null!;
    public Guid UserId { get; set; }
    public DateTime ExpiresAt { get; set; }
}