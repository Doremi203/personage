namespace Personage.Auth.Domain.Exceptions;

public class OAuthException(string message) : CustomException(message)
{
    public override ErrorCode ErrorCode => ErrorCode.OAuthError;
}