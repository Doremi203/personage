using System.Data;

namespace Personage.Auth.DataAccess.Interfaces;

public interface IDbConnectionFactory
{
    Task<IDbConnection> CreateConnection(CancellationToken token);
}