namespace Personage.Auth.Domain.Exceptions.Base;

public class ValidationException(ErrorCode errorCode, string message) : CustomException(errorCode, message);