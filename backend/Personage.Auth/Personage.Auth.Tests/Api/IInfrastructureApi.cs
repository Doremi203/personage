using RestEase;

namespace Personage.Auth.Tests.Api;


public interface IInfrastructureApi
{
    [Get("liveness")]
    Task Liveness();

    [Get("health")]
    Task Health();
}