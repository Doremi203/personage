namespace Personage.Auth.Domain.Exceptions;

public abstract class CustomException : Exception
{
    protected CustomException(string message) : base(message)
    {
    }

    protected CustomException(string message, Exception innerException) : base(message, innerException)
    {
    }

    public ErrorCode ErrorCode { get; set; }
}