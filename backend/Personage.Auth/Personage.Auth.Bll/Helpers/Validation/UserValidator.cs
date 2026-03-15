using System.Text.RegularExpressions;
using Personage.Auth.Domain.Exceptions.Base;

namespace Personage.Auth.Bll.Helpers.Validation;

public static partial class UserValidator
{
    private const string EmailPattern = @"^[^@\s]+@[^@\s]+\.[^@\s]+$";
    private const string NamePattern = @"^[\p{L}\s\-']+$";
    private const string PasswordPattern = @"^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)[A-Za-z\d\W_]{8,}$";
    // (?=.*[a-z]) - at least one lowercase letter
    // (?=.*[A-Z]) - at least one uppercase letter  
    // (?=.*\d)   - at least one digit
    // [A-Za-z\d\W_]{8,} - at least 8 characters from allowed set (letters, digits, special characters)
    
    [GeneratedRegex(EmailPattern, RegexOptions.IgnoreCase | RegexOptions.ExplicitCapture)]
    private static partial Regex EmailRegex();
    
    [GeneratedRegex(NamePattern)]
    private static partial Regex NameRegex();
    
    [GeneratedRegex(PasswordPattern)]
    private static partial Regex PasswordRegex();
    
    public static void ValidateUser(string email, string password, string name)
    {
        ValidateEmail(email);
        ValidatePassword(password);
        ValidateName(name);
    }

    public static void ValidateEmail(string email)
    {
        if (string.IsNullOrWhiteSpace(email))
            throw new ValidationException(ErrorCode.EmailValidationFail, "Email cannot be empty");

        if (!EmailRegex().IsMatch(email))
            throw new ValidationException(ErrorCode.EmailValidationFail, "Invalid email. Your email should have the following format: email@example.com");
    }
    
    public static void ValidatePassword(string password)
    {
        if (string.IsNullOrWhiteSpace(password))
            throw new ValidationException(ErrorCode.PasswordValidationFail, "Password cannot be empty");
        
        if (password.Length < 8)
            throw new ValidationException(ErrorCode.PasswordValidationFail, "Password must be at least 8 characters long");
        
        if (!PasswordRegex().IsMatch(password))
        {
            throw new ValidationException(ErrorCode.PasswordValidationFail, 
                "Password must contain at least one lowercase letter, one uppercase letter, and one digit");
        }
    }
    
    private static void ValidateName(string name)
    {
        if (string.IsNullOrWhiteSpace(name))
            throw new ValidationException(ErrorCode.UserNameValidationFail, "Name cannot be empty");
        
        switch (name.Length)
        {
            case < 2:
                throw new ValidationException(ErrorCode.UserNameValidationFail, "Name must be at least 2 characters long");
            case > 100:
                throw new ValidationException(ErrorCode.UserNameValidationFail, "Name cannot exceed 100 characters");
        }

        if (!NameRegex().IsMatch(name))
        {
            throw new ValidationException(
                ErrorCode.UserNameValidationFail,
                "Name can only contain letters, spaces, hyphens, and apostrophes"
            );
        }
    }
}