using System.Threading;
using System.Threading.Tasks;
using Personage.Auth.Api.Contracts.Test.Responses;
using RestEase;

namespace Personage.Auth.Tests.Api;

[BasePath("test")]
public interface ITestApi
{
    [Get("ping")]
    Task<PingResponse> PingAsync(CancellationToken ct);
}