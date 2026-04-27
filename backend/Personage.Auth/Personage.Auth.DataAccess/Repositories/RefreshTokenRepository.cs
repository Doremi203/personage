using Dapper;
using Personage.Auth.DataAccess.Interfaces;
using Personage.Auth.DataAccess.Interfaces.Repositories;
using Personage.Auth.DataAccess.Models;
using Personage.Auth.DataAccess.Models.Requests;

namespace Personage.Auth.DataAccess.Repositories;

public class RefreshTokenRepository(IDbConnectionFactory connectionFactory) : IRefreshTokenRepository
{
    public async Task<RefreshToken> CreateRefreshToken(CreateRefreshTokenRequest request, CancellationToken ct)
    {
        using var connection = await connectionFactory.CreateConnection(ct);

        return await connection.QueryFirstAsync<RefreshToken>(
            """
            --RefreshTokenRepository.CreateRefreshToken
            INSERT INTO refresh_token (token, user_id, expires_at) 
            VALUES (@token, @user_id, @expires_at)
            RETURNING
                id,
                token,
                user_id as UserId,
                created_at as CreatedAt,
                expires_at as ExpiresAt;
            """,
            new
            {
                token = request.Token,
                user_id = request.UserId,
                expires_at = request.ExpiresAt
            });
    }

    public async Task<RefreshToken?> GetRefreshToken(string token, CancellationToken ct)
    {
        using var connection = await connectionFactory.CreateConnection(ct);

        return await connection.QueryFirstOrDefaultAsync<RefreshToken>(
            """
             --RefreshTokenRepository.GetRefreshToken
             SELECT
                rt.id,
                rt.token,
                rt.user_id as UserId,
                rt.created_at as CreatedAt,
                rt.expires_at as ExpiresAt
             FROM refresh_token rt
             WHERE token=@token;
             """,
            new { token });
    }
}