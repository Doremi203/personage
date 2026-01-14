using System;
using System.Threading;
using System.Threading.Tasks;
using AutoFixture;
using Dapper;
using Personage.Auth.DataAccess.Interfaces;
using Personage.Auth.DataAccess.Models;

namespace Personage.Auth.Tests.Infrastructure.Repositories;

public class TestUserRepository(IDbConnectionFactory connectionFactory)
{
    private static Fixture Fixture { get; } = new Fixture();

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

    public async Task<(Guid UserId, GmailToken Token)> CreateUserWithToken(DateTime? userProcessedAt)
    {
        var userId = await CreateUser();
        using var connection = await connectionFactory.CreateConnection(CancellationToken.None);

        var token = await connection.QueryFirstAsync<GmailToken>(
            """
            INSERT INTO gmail_token(user_id, access_token, refresh_token, expires_at, gmail_email, last_processed_at)
            VALUES (@userId, @accessToken, @refreshToken, @expiresAt, @gmailEmail, @userProcessedAt)
            returning
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
                expiresAt = DateTime.UtcNow.AddMinutes(Random.Shared.Next(10, 20)),
                gmailEmail = Fixture.Create<string>(),
                userProcessedAt
            });
        return (userId, token);
    }
}