using Dapper;
using Personage.Auth.DataAccess.Interfaces;
using Personage.Auth.DataAccess.Interfaces.Repositories;
using Personage.Auth.DataAccess.Models;

namespace Personage.Auth.DataAccess.Repositories;

public class UserRepository(IDbConnectionFactory connectionFactory) : IUserRepository
{
    public async Task<User?> GetUserAsync(string userId, CancellationToken ct)
    {
        var connection = await connectionFactory.CreateConnection(ct);
        
        return await connection.QueryFirstOrDefaultAsync<User>(
            """
            SELECT 
                id, 
                email, 
                created_at as CreatedAt, 
                updated_at as UpdatedAt 
            FROM "User" 
            WHERE id = @userId
            """,
            new { userId });
    }

    public async Task<User> CreateOrUpdateUserAsync(string userId, string email, CancellationToken ct)
    {
        var connection = await connectionFactory.CreateConnection(ct);
        
        return await connection.QueryFirstAsync<User>(
            """
            SELECT 
                id, 
                email, 
                created_at as CreatedAt, 
                updated_at as UpdatedAt 
            FROM "user" 
            WHERE id = @userId
            """,
            new { userId });
    }
}