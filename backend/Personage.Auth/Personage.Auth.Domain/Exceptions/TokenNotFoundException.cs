namespace Personage.Auth.Domain.Exceptions;

public class TokenNotFoundException(string message) : CustomException(message)
{
    public override ErrorCode ErrorCode => ErrorCode.TokenNotFound;
}