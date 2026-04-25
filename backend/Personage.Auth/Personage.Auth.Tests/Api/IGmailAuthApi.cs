using Personage.Auth.Api.Contracts.Auth.OAuth.Requests;
using Personage.Auth.Api.Contracts.Auth.OAuth.Responses;
using RestEase;

namespace Personage.Auth.Tests.Api;

[BasePath("auth/gmail")]
public interface IGmailAuthApi
{
    [Post("authorize")]
    Task<StartAuthResponse> StartGmailAuth([Body] StartAuthRequest request, CancellationToken ct);

    [Post("callback")]
    Task<AuthCallbackResponse> HandleGmailCallback([Body] AuthCallbackRequest request, CancellationToken ct);
}