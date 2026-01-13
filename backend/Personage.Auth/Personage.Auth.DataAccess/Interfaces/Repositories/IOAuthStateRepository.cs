using Personage.Auth.DataAccess.Models;

namespace Personage.Auth.DataAccess.Interfaces.Repositories;

public interface IOAuthStateRepository
{
    Task<OAuthState?> GetState(string state, CancellationToken ct);
    Task SaveState(OAuthState state, CancellationToken ct);
    Task DeleteState(string state, CancellationToken ct);
}