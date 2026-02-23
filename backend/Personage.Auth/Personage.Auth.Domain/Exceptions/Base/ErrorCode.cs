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
    InvalidCredentials = 6,
    InvalidRefreshToken = 7,
    UserAlreadyExists = 8,
    EmailValidationFail = 9,
    PasswordValidationFail = 10,
    UserNameValidationFail = 11,
    PasswordNotSet = 12
}