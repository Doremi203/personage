using Personage.Auth.DataAccess.Models;

namespace Personage.Auth.DataAccess.Interfaces.Repositories;

public interface IGoogleCalendarTokenRepository
{
    Task SaveToken(OAuthToken token, CancellationToken ct);
}