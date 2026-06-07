using AutoFixture;
using Dapper;
using Personage.Auth.DataAccess.Interfaces;
using Personage.Auth.DataAccess.Models;

namespace Personage.Auth.Tests.Infrastructure.Repositories;

public class TestUserRepository(IDbConnectionFactory connectionFactory)
{
    private static Fixture Fixture { get; } = new();

    public async Task<Guid> CreateUser(string? email = null)
    {
        email ??= Fixture.Create<string>();

        using var connection = await connectionFactory.CreateConnection(CancellationToken.None);
        return await connection.QueryFirstAsync<Guid>(
            """
            INSERT INTO "user"(email)
            VALUES(@email)
            RETURNING id;
            """,
            new { email });
    }

    public async Task<(Guid UserId, OAuthToken Token)> CreateUserWithToken(
        DateTime? userProcessedAt,
        DateTime? expiresAt = null)
    {
        var userId = await CreateUser();
        using var connection = await connectionFactory.CreateConnection(CancellationToken.None);

        var token = await connection.QueryFirstAsync<OAuthToken>(
            """
            INSERT INTO gmail_token(user_id, access_token, refresh_token, expires_at, gmail_email, last_processed_at)
            VALUES (@userId, @accessToken, @refreshToken, @expiresAt, @gmailEmail, @userProcessedAt)
            returning
                id as TokenId,
                user_id as UserId,
                access_token as AccessToken,
                refresh_token as RefreshToken,
                expires_at as ExpiresAt,
                gmail_email as GmailEmail
            """, new
            {
                userId,
                accessToken = Fixture.Create<string>(),
                refreshToken = Fixture.Create<string>(),
                expiresAt = expiresAt ?? DateTime.UtcNow.AddMinutes(Random.Shared.Next(10, 20)),
                gmailEmail = Fixture.Create<string>(),
                userProcessedAt
            });
        return (userId, token);
    }

    public async Task<(bool Exists, int FailedRefreshes)> GetGmailTokenState(Guid userId)
    {
        using var connection = await connectionFactory.CreateConnection(CancellationToken.None);

        var failedRefreshes = await connection.QueryFirstOrDefaultAsync<int?>(
            """
            SELECT failed_refreshes
            FROM gmail_token
            WHERE user_id = @userId;
            """,
            new { userId });

        return (failedRefreshes.HasValue, failedRefreshes ?? 0);
    }
}