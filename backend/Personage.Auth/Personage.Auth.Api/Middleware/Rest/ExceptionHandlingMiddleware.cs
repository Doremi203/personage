using System.Text.Json;
using Personage.Auth.Api.Contracts.Common;
using Personage.Auth.Domain.Exceptions.Base;

namespace Personage.Auth.Api.Middleware.Rest;

public class ExceptionHandlingMiddleware(RequestDelegate next)
{
    private readonly JsonSerializerOptions _jsonOptions = new()
    {
        PropertyNamingPolicy = JsonNamingPolicy.CamelCase
    };

    public async Task InvokeAsync(HttpContext context)
    {
        try
        {
            await next(context);
        }
        catch (Exception ex)
        {
            await HandleExceptionAsync(context, ex);
        }
    }

    private async Task HandleExceptionAsync(HttpContext context, Exception exception)
    {
        var (statusCode, errorCode, message) = GetExceptionDetails(exception);

        var errorResponse = new ErrorResponse(errorCode, message, statusCode);

        context.Response.ContentType = "application/json";
        context.Response.StatusCode = statusCode;

        await context.Response.WriteAsync(JsonSerializer.Serialize(errorResponse, _jsonOptions));
    }

    private (int StatusCode, ErrorCode ErrorCode, string Message) GetExceptionDetails(Exception exception)
    {
        return exception switch
        {
            CustomException customEx => HandleCustomException(customEx),
            _ => (StatusCodes.Status500InternalServerError, ErrorCode.Unknown, "An unexpected error occurred")
        };
    }

    private (int StatusCode, ErrorCode ErrorCode, string Message) HandleCustomException(CustomException customEx)
    {
        var statusCode = GetHttpStatusCode(customEx);
        var message = customEx.Message;
        if(customEx is AuthenticationException)
            message = "Authentication exception. Please log in again.";
        
        return (
            statusCode,
            customEx.ErrorCode,
            message
        );
    }

    private static int GetHttpStatusCode(CustomException customException)
    {
        return customException switch
        {
            NotFoundException => StatusCodes.Status404NotFound,
            AuthenticationException => StatusCodes.Status401Unauthorized,
            ValidationException => StatusCodes.Status400BadRequest,
            _ => customException.ErrorCode switch
            {
                ErrorCode.OAuthError => StatusCodes.Status400BadRequest,
                ErrorCode.InvalidRefreshToken => StatusCodes.Status400BadRequest,
                ErrorCode.InvalidResetToken => StatusCodes.Status400BadRequest,
                ErrorCode.EmailValidationFail => StatusCodes.Status400BadRequest,
                ErrorCode.PasswordValidationFail => StatusCodes.Status400BadRequest,
                ErrorCode.UserNameValidationFail => StatusCodes.Status400BadRequest,

                ErrorCode.InvalidCredentials => StatusCodes.Status401Unauthorized,
                ErrorCode.PasswordNotSet => StatusCodes.Status401Unauthorized,

                ErrorCode.TokenNotFound => StatusCodes.Status404NotFound,
                ErrorCode.TelegramSessionNotFound => StatusCodes.Status404NotFound,

                ErrorCode.UserAlreadyExists => StatusCodes.Status409Conflict,

                ErrorCode.DuplicatedUsersForbidden => StatusCodes.Status500InternalServerError,
                ErrorCode.UsersNotAuthorizedForProcessing => StatusCodes.Status500InternalServerError,

                // Unimplemented (501)
                ErrorCode.ServiceTypeNotSupported => StatusCodes.Status501NotImplemented,

                // Default
                _ => StatusCodes.Status500InternalServerError,
            }
        };
    }
}