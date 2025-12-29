using System.Threading;
using System.Threading.Tasks;
using Personage.Auth.Contracts.Test.Responses;
using RestEase;

namespace Personage.Auth.Tests.Api;

[BasePath("test")]
public interface ITestApi
{
    [Get("ping")]
    Task<PingResponse> PingAsync(CancellationToken ct);
}