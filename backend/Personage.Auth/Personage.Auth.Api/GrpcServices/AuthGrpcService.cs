using Google.Protobuf.WellKnownTypes;
using Grpc.Core;
using Personage.Auth.Api.Grpc;
using Personage.Auth.Domain.Interfaces;

namespace Personage.Auth.Api.GrpcServices;

public class AuthGrpcService(IAuthService authService) : AuthService.AuthServiceBase
{
    public override async Task<GmailTokens> GetGmailTokens(GetGmailTokensRequest request, ServerCallContext context)
    {
        var res = await authService.GetUserGmailToken(request.UserEmail, context.CancellationToken);
        return new GmailTokens
        {
            AccessToken = res.AccessToken,
            RefreshToken = res.RefreshToken,
            ExpiresAt = Timestamp.FromDateTime(res.ExpiresAt)
        };
    }
}