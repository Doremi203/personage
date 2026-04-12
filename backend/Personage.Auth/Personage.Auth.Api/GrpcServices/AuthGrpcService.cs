using Google.Protobuf.WellKnownTypes;
using Grpc.Core;
using Personage.Auth.Api.Grpc;
using Personage.Auth.Api.Grpc.Common;
using Personage.Auth.Domain.Interfaces;

namespace Personage.Auth.Api.GrpcServices;

public class AuthGrpcService(ITokenService tokenService) : AuthService.AuthServiceBase
{
    public override async Task<GmailTokens> GetGmailTokens(GetGmailTokensRequest request, ServerCallContext context)
    {
        var res = await tokenService.GetUserGmailToken(request.UserEmail, context.CancellationToken);
        return new GmailTokens
        {
            AccessToken = res.AccessToken,
            RefreshToken = res.RefreshToken,
            ExpiresAt = Timestamp.FromDateTime(res.ExpiresAt),
            GmailEmail = res.GmailEmail
        };
    }

    public override Task<VerifyTokenResponse> VerifyToken(VerifyTokenRequest request, ServerCallContext context)
    {
        var isValid = tokenService.VerifyToken(request.Token);
        return Task.FromResult(new VerifyTokenResponse
        {
            IsValid = isValid
        });
    }
}