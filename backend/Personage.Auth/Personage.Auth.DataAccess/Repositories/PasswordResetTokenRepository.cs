using Dapper;
using Personage.Auth.DataAccess.Interfaces;
using Personage.Auth.DataAccess.Interfaces.Repositories;
using Personage.Auth.DataAccess.Models;

namespace Personage.Auth.DataAccess.Repositories;

public class PasswordResetTokenRepository(IDbConnectionFactory connectionFactory) : IPasswordResetTokenRepository
{
    public async Task<PasswordResetToken?> GetToken(string token, CancellationToken ct)
    {
        using var connection = await connectionFactory.CreateConnection(ct);

        return await connection.QueryFirstOrDefaultAsync<PasswordResetToken>(
            """
            SELECT
                id,
                user_id as UserId,
                token as Token,
                expires_at as ExpiresAt,
                used_at as UsedAt
            FROM password_reset_token
            WHERE token = @token AND used_at IS NULL AND expires_at > NOW()
            """,
            new { token });
    }

    public async Task<PasswordResetToken> CreateToken(Guid userId, string token, DateTime expiresAt,
        CancellationToken ct)
    {
        using var connection = await connectionFactory.CreateConnection(ct);

        var id = Guid.NewGuid();
        await connection.ExecuteAsync(
            """
            INSERT INTO password_reset_token (id, user_id, token, expires_at, created_at) 
            VALUES (@Id, @UserId, @Token, @ExpiresAt, NOW())
            """,
            new
            {
                Id = id,
                UserId = userId,
                Token = token,
                ExpiresAt = expiresAt
            });

        return new PasswordResetToken
        {
            Id = id,
            UserId = userId,
            Token = token,
            ExpiresAt = expiresAt,
            CreatedAt = DateTime.UtcNow
        };
    }

    public async Task InvalidateToken(string token, CancellationToken ct)
    {
        using var connection = await connectionFactory.CreateConnection(ct);
        await connection.ExecuteAsync(
            "UPDATE password_reset_token SET used_at = NOW() WHERE token = @token",
            new { token });
    }

    public async Task InvalidateAllUserTokens(Guid userId, CancellationToken ct)
    {
        using var connection = await connectionFactory.CreateConnection(ct);
        await connection.ExecuteAsync(
            "UPDATE password_reset_token SET used_at = NOW() WHERE user_id = @userId AND used_at IS NULL",
            new { userId });
    }
}