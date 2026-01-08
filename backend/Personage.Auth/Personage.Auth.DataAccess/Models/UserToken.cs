namespace Personage.Auth.DataAccess.Models;

public class UserToken
{
    public int Id { get; set; }
    public string UserId { get; set; } = null!;
    public ServiceType ServiceType { get; set; }
    public string AccessToken { get; set; } = null!;
    public string RefreshToken { get; set; } = null!;
}