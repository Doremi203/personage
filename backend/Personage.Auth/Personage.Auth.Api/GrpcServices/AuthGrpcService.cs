using Google.Protobuf.WellKnownTypes;
using Grpc.Core;
using Personage.Auth.Api.Grpc;
using Personage.Auth.Api.Grpc.Common;
using Personage.Auth.Domain.Interfaces;

namespace Personage.Auth.Api.GrpcServices;

public class AuthGrpcService(
    ITokenService tokenService,
    IUserProfileService userProfileService
) : AuthService.AuthServiceBase
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

    public override async Task<UserProfile> GetUserProfile(GetUserProfileRequest request, ServerCallContext context)
    {
        if (!Guid.TryParse(request.UserId, out var userId))
            throw new RpcException(new Status(StatusCode.InvalidArgument, "user_id must be a UUID"));

        var profile = await userProfileService.GetUserById(userId, context.CancellationToken);

        return new UserProfile
        {
            UserId = profile.Id.ToString(),
            Email = profile.Email,
            Name = profile.Name ?? string.Empty,
            CreatedAt = Timestamp.FromDateTime(DateTime.SpecifyKind(profile.CreatedAt, DateTimeKind.Utc)),
        };
    }
}
