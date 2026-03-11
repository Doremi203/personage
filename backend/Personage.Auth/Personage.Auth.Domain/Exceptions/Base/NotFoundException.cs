namespace Personage.Auth.Domain.Exceptions.Base;

public class NotFoundException : CustomException
{
    public NotFoundException(string message) : base(message)
    {
    }

    public NotFoundException(ErrorCode errorCode, string message) : base(errorCode, message)
    {
    }
}
