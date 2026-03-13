using Grpc.Core;
using Grpc.Core.Interceptors;
using Personage.Auth.Domain.Exceptions.Base;

namespace Personage.Auth.Api.Middleware;


public class ExceptionInterceptor(ILogger<ExceptionInterceptor> logger) : Interceptor
{
    public override async Task<TResponse> UnaryServerHandler<TRequest, TResponse>(
        TRequest request,
        ServerCallContext context,
        UnaryServerMethod<TRequest, TResponse> continuation)
    {
        try
        {
            return await continuation(request, context);
        }
        catch (CustomException customEx)
        {
            var statusCode = GetGrpcStatusCode(customEx);
            var logLevel = GetLogLevel(statusCode);
            
            logger.Log(logLevel, customEx, "Domain exception: {ErrorCode} - {Message}", 
                customEx.ErrorCode, customEx.Message);
            
            throw new RpcException(new Status(statusCode, customEx.Message));
        }
        catch (RpcException)
        {
            throw;
        }
        catch (Exception ex)
        {
            logger.LogError(ex, "Unhandled exception in gRPC call");
            throw new RpcException(new Status(StatusCode.Internal, "Internal server error"));
        }
    }


    private static StatusCode GetGrpcStatusCode(CustomException customException)
    {
        if(customException is NotFoundException)
            return StatusCode.NotFound;
        
        return customException.ErrorCode switch
        {
            ErrorCode.TokenNotFound => StatusCode.NotFound,
            ErrorCode.OAuthError => StatusCode.InvalidArgument,
            ErrorCode.ServiceTypeNotSupported => StatusCode.Unimplemented,
            ErrorCode.DuplicatedUsersForbidden => StatusCode.InvalidArgument,
            ErrorCode.UsersNotAuthorizedForProcessing => StatusCode.InvalidArgument,
            _ => StatusCode.Unknown
        };
    }
    
    private static LogLevel GetLogLevel(StatusCode statusCode)
    {
        return statusCode switch
        {
            StatusCode.NotFound => LogLevel.Warning,
            StatusCode.InvalidArgument => LogLevel.Warning,
            StatusCode.Unauthenticated => LogLevel.Warning,
            StatusCode.PermissionDenied => LogLevel.Warning,
            StatusCode.AlreadyExists => LogLevel.Warning,
            StatusCode.ResourceExhausted => LogLevel.Warning,
            _ => LogLevel.Error
        };
    }
}