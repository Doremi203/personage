using System.Text.Json.Serialization;

namespace Personage.Auth.Domain.Exceptions.Base;

[JsonConverter(typeof(JsonStringEnumConverter))]
public enum ErrorCode
{
    Unknown = 0,
    OAuthError = 1,
    TokenNotFound = 2,
    ServiceTypeNotSupported = 3,
    DuplicatedUsersForbidden = 4,
    UsersNotAuthorizedForProcessing = 5,
    UserNotFound = 6,
    InvalidRefreshToken = 7,
    UserAlreadyExists = 8,
    InvalidEmail = 9,
    InvalidPassword = 10,
    InvalidName = 11,
}