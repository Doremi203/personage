using Personage.Auth.DataAccess.Models;
using Personage.Auth.DataAccess.Models.Requests;

namespace Personage.Auth.DataAccess.Interfaces.Repositories;

public interface IRefreshTokenRepository
{
    Task<RefreshToken> CreateRefreshToken(CreateRefreshTokenRequest request, CancellationToken ct);
    Task<RefreshToken?> GetRefreshToken(string token, CancellationToken ct);
}