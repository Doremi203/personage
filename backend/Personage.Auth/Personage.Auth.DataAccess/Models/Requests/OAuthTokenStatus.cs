namespace Personage.Auth.DataAccess.Models.Requests;

public enum OAuthTokenStatus
{
    Unknown = 0,
    Active = 1,
    Expired = 2,
    Invalid = 3,
    Revoked = 4
}