namespace Personage.Auth.Domain.Exceptions;

public class OAuthException : CustomException
{
    public OAuthException(string message) : base(message)
    {
        ErrorCode = ErrorCode.OAuthError;
    }
    
    public OAuthException(string message, Exception innerException) 
        : base(message, innerException)
    {
        ErrorCode = ErrorCode.OAuthError;
    }
}