using Google.Protobuf.WellKnownTypes;
using Personage.Auth.Api.Grpc.Common;
using Personage.Auth.Domain.Models.Common;

namespace Personage.Auth.Api.Mappers;

public static class CommonMapper
{
    public static GmailTokens ToGrpcGmailTokens(GmailTokenModel value)
    {
        return new GmailTokens
        {
            AccessToken = value.AccessToken,
            RefreshToken = value.RefreshToken,
            ExpiresAt = Timestamp.FromDateTime(value.ExpiresAt),
            GmailEmail = value.GmailEmail
        };
    }
}