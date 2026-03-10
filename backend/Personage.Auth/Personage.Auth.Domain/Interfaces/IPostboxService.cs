namespace Personage.Auth.Domain.Interfaces;

public interface IPostboxService
{
    Task SendPasswordResetEmail(string email, string resetUrl, CancellationToken ct);
}