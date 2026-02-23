namespace Personage.Auth.Domain.Exceptions.Base;

public class AuthenticationException(ErrorCode errorCode, string message) : CustomException(errorCode, message);