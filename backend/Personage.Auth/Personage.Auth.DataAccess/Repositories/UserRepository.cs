using Dapper;
using Personage.Auth.DataAccess.Interfaces;
using Personage.Auth.DataAccess.Interfaces.Repositories;
using Personage.Auth.DataAccess.Models;

namespace Personage.Auth.DataAccess.Repositories;

public class UserRepository(IDbConnectionFactory connectionFactory) : IUserRepository
{
    public async Task<User?> GetUserByEmail(string email, CancellationToken ct)
    {
        using var connection = await connectionFactory.CreateConnection(ct);
        
        return await connection.QueryFirstOrDefaultAsync<User>(
            """
            SELECT 
                id, 
                email, 
                created_at as CreatedAt
            FROM "User" 
            WHERE email = @email;
            """,
            new { email });
    }

    public async Task<User> CreateUser(string email, CancellationToken ct)
    {
        using var connection = await connectionFactory.CreateConnection(ct);
        
        return await connection.QuerySingleAsync<User>(
            """
            INSERT INTO "user" (email) 
            VALUES (@email) 
            RETURNING 
                id,
                email,
                created_at as CreatedAt;
            """,
            new { email });
    }
}