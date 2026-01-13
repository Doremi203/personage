using Personage.Auth.DataAccess.Models;

namespace Personage.Auth.DataAccess.Interfaces.Repositories;

public interface IGmailTokenRepository
{
    Task<GmailToken?> GetTokenByUserEmail(string userEmail, CancellationToken ct);
    Task SaveToken(GmailToken token, CancellationToken ct);
}