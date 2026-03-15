using Personage.Auth.Domain.Exceptions.Base;

namespace Personage.Auth.Api.Contracts.Common;

public class ErrorResponse(ErrorCode errorCode, string message, int statusCode)
{
    public ErrorCode ErrorCode { get; set; } = errorCode;
    public string Message { get; set; } = message;
    public int StatusCode { get; set; } = statusCode;
}