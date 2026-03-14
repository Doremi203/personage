using Dapper;
using Personage.Auth.DataAccess.Interfaces;
using Personage.Auth.DataAccess.Interfaces.Repositories;
using Personage.Auth.DataAccess.Models;
using Personage.Auth.DataAccess.Models.Requests;

namespace Personage.Auth.DataAccess.Repositories;

public class UserRepository(IDbConnectionFactory connectionFactory) : IUserRepository
{
    private const string UserFields =
        """
        u.id as Id,
        u.email as Email,
        u.name as Name,
        u.created_at as CreatedAt,
        u.password_hash as PasswordHash
        """;
    public async Task<User?> GetUserByEmail(string email, CancellationToken ct)
    {
        using var connection = await connectionFactory.CreateConnection(ct);
        
        return await connection.QueryFirstOrDefaultAsync<User>(
            $"""
            --UserRepository.GetUserByEmail
            SELECT
            {UserFields}
            FROM "user" u
            WHERE email = @email;
            """,
            new { email });
    }

    public async Task<User?> GetUserById(Guid userId, CancellationToken ct)
    {
        using var connection = await connectionFactory.CreateConnection(ct);
        
        return await connection.QueryFirstOrDefaultAsync<User>(
            $"""
            --UserRepository.GetUserById
            SELECT
            {UserFields}
            FROM "user" u
            WHERE id = @userId;
            """,
            new { userId });
    }

    public async Task<User> CreateShortUser(string email, CancellationToken ct)
    {
        using var connection = await connectionFactory.CreateConnection(ct);
        
        return await connection.QuerySingleAsync<User>(
            """
            --UserRepository.CreateShortUser
            INSERT INTO "user" (email) 
            VALUES (@email) 
            RETURNING 
                id,
                email,
                name,
                created_at as CreatedAt,
                password_hash as PasswordHash;
            """,
            new { email });
    }

    public async Task<User> CreateUser(CreateUserRequest request, CancellationToken ct)
    {
        using var connection = await connectionFactory.CreateConnection(ct);

        return await connection.QuerySingleAsync<User>(
            """
            --UserRepository.CreateUser
            INSERT INTO "user" (email, name, password_hash) 
            VALUES (@email, @name, @password_hash) 
            RETURNING 
                id,
                email,
                name,
                created_at as CreatedAt,
                password_hash as PasswordHash;
            """,
            new
            {
                email = request.Email, 
                name = request.Name, 
                password_hash = request.PasswordHash
            });
    }

    public async Task<UserWithToken[]> GetUsersGmailProcessedBeforeMoment(
        DateTime processedBeforeMoment, 
        int limit, 
        CancellationToken ct)
    {
        using var connection = await connectionFactory.CreateConnection(ct);
    
        const string query = 
            """
            --UserRepository.GetUsersGmailProcessedBeforeMoment   
            SELECT u.id AS UserId,
                u.email AS UserEmail,
                gt.Id as TokenId,
                gt.access_token AS AccessToken,
                gt.refresh_token AS RefreshToken,
                gt.expires_at AS ExpiresAt,
                gt.gmail_email AS GmailEmail,
                gt.last_processed_at AS LastProcessedAt
            FROM public.user u
            JOIN public.gmail_token gt ON u.id = gt.user_id
            WHERE 
                gt.last_processed_at IS NULL OR 
                gt.last_processed_at <= @processedBeforeMoment
            ORDER BY 
                gt.last_processed_at ASC NULLS FIRST
            LIMIT @limit;
            """;

        var results = await connection.QueryAsync<UserWithToken, ShortGmailToken, UserWithToken>(
            query,
            (user, token) =>
            {
                user.Token = token;
                return user;
            },
            new
            {
                processedBeforeMoment,
                limit
            },
            splitOn: nameof(ShortGmailToken.TokenId)
        );

        return results.ToArray();
    }

    public async Task<UserWithTelegramSession[]> GetUsersTelegramProcessedBeforeMoment(DateTime processedBeforeMoment, int limit, CancellationToken ct)
    {
        using var connection = await connectionFactory.CreateConnection(ct);
        
        var results = await connection.QueryAsync<UserWithTelegramSession>(
            """
            --UserRepository.GetUsersTelegramProcessedBeforeMoment   
            SELECT 
                ts.user_id AS UserId,
                ts.last_processed_at AS LastProcessedAt,
                ts.session AS SessionString
            FROM public.telegram_session ts
            WHERE 
                ts.last_processed_at IS NULL OR 
                ts.last_processed_at <= @processedBeforeMoment
            ORDER BY 
                ts.last_processed_at ASC NULLS FIRST
            LIMIT @limit;
            """,
            new
            {
                processedBeforeMoment,
                limit
            }
        );

        return results.ToArray();
    }


    public async Task UpdatePassword(Guid userId, string passwordHash, CancellationToken ct)
    {
        using var connection = await connectionFactory.CreateConnection(ct);
        
        await connection.ExecuteAsync(
            """
            --UserRepository.UpdatePassword
            UPDATE "user"
            SET password_hash = @passwordHash
            WHERE id = @userId;
            """,
            new
            {
                userId,
                passwordHash
            });
    }
}