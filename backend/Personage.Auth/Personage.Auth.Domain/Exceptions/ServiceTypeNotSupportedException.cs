namespace Personage.Auth.Domain.Exceptions;

public class ServiceTypeNotSupportedException(string message) : CustomException(message)
{
    public override ErrorCode ErrorCode => ErrorCode.ServiceTypeNotSupported;
}