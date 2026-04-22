using Google.Protobuf.WellKnownTypes;
using Personage.Auth.Api.Grpc.Common;
using Personage.Auth.Domain.Models.Common;

namespace Personage.Auth.Api.Mappers;

public static class CommonMapper
{
    public static GmailTokens? ToGrpcGmailTokens(OAuthTokenModel? value)
    {
        if (value is null)
            return null;
        
        return new GmailTokens
        {
            AccessToken = value.AccessToken,
            RefreshToken = value.RefreshToken,
            ExpiresAt = Timestamp.FromDateTime(value.ExpiresAt),
            GmailEmail = value.GmailEmail
        };
    }   
    
    public static GoogleCalendarTokens? ToGrpcGoogleCalendarTokens(OAuthTokenModel? value)
    {
        if (value is null)
            return null;
        
        return new GoogleCalendarTokens
        {
            AccessToken = value.AccessToken,
            RefreshToken = value.RefreshToken,
            ExpiresAt = Timestamp.FromDateTime(value.ExpiresAt),
            GmailEmail = value.GmailEmail
        };
    }

    public static ServiceTypeModel ToDomainServiceType(ServiceType value)
    {
        return value switch
        {
            ServiceType.Unknown => ServiceTypeModel.Unknown,
            ServiceType.Gmail => ServiceTypeModel.Gmail,
            ServiceType.Telegram => ServiceTypeModel.Telegram,
            ServiceType.GoogleCalendar => ServiceTypeModel.GoogleCalendar,
            _ => ServiceTypeModel.Unknown
        };
    }
}