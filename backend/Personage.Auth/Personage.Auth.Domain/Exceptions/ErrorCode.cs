using System.Text.Json.Serialization;

namespace Personage.Auth.Domain.Exceptions;

[JsonConverter(typeof(JsonStringEnumConverter))]
public enum ErrorCode
{
    Unknown = 0,
    OAuthError = 1,
    TokenNotFound = 2,
}