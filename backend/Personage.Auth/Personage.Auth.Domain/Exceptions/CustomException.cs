namespace Personage.Auth.Domain.Exceptions;

public class CustomException : Exception
{
    protected CustomException(string message) : base(message)
    {
    }

    public CustomException(ErrorCode errorCode, string message) : base(message)
    {
        ErrorCode = errorCode;
    }

    public virtual ErrorCode ErrorCode { get; }
}