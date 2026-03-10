using Personage.Auth.Domain.Exceptions.Base;

namespace Personage.Auth.Domain.Exceptions;

public class TokenNotFoundException(string message) : NotFoundException(message)
{
    public override ErrorCode ErrorCode => ErrorCode.TokenNotFound;
}