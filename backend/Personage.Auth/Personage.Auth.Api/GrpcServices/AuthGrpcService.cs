using Grpc.Core;
using Personage.Auth.Api.Grpc;

namespace Personage.Auth.GrpcServices;

public class AuthGrpcService : AuthService.AuthServiceBase
{
    public override Task<GmailTokens> GetGmailTokens(GetGmailTokensRequest request, ServerCallContext context)
    {
        throw new NotImplementedException();
    }

    public override Task<HasGmailAccessResponse> HasGmailAccess(HasGmailAccessRequest request, ServerCallContext context)
    {
        throw new NotImplementedException();
    }

    public override Task<GoogleAuthUrl> GetGoogleAuthUrl(GetGoogleAuthUrlRequest request, ServerCallContext context)
    {
        throw new NotImplementedException();
    }

    public override Task<AuthResponse> HandleGoogleAuthCallback(GoogleAuthCallbackRequest request, ServerCallContext context)
    {
        throw new NotImplementedException();
    }
}