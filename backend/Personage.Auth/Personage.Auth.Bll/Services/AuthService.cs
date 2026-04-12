using System.Security.Cryptography;
using Microsoft.Extensions.Logging;
using Personage.Auth.Bll.Helpers;
using Personage.Auth.Bll.Helpers.Validation;
using Personage.Auth.DataAccess.Interfaces.Repositories;
using Personage.Auth.DataAccess.Models;
using Personage.Auth.DataAccess.Models.Requests;
using Personage.Auth.Domain.Exceptions;
using Personage.Auth.Domain.Exceptions.Base;
using Personage.Auth.Domain.Interfaces;
using Personage.Auth.Domain.Models.Auth.Requests;
using Personage.Auth.Domain.Models.Common;

namespace Personage.Auth.Bll.Services;

public class AuthService(
    IUserRepository userRepository,
    IGmailTokenRepository gmailTokenRepository,
    IOAuthStateRepository oauthStateRepository,
    IPasswordResetTokenRepository passwordResetTokenRepository,
    IGoogleOAuthService googleOAuthService,
    IPostboxService postboxService,
    ITokenService tokenService,
    ILogger<AuthService> logger
) : IAuthService
{
    public async Task<StartGmailAuthResponseModel> StartGmailAuth(string userEmail, string redirectUri,
        CancellationToken ct)
    {
        UserValidator.ValidateEmail(userEmail);
        //get user id from claims here and use it to check user and assign tokens
        if (await userRepository.GetUserByEmail(userEmail, ct) is null)
        {
            //TODO: update flow, do not allow for gmail linking without an active account
            logger.LogWarning("User with email {Email} not found, creating new user", userEmail);
            await userRepository.CreateShortUser(userEmail, ct);
        }

        var state = GenerateState();

        var oauthState = new OAuthState
        {
            State = state,
            UserEmail = userEmail,
            RedirectUri = redirectUri,
            ExpiresAt = DateTime.UtcNow.AddMinutes(10)
        };

        await oauthStateRepository.SaveState(oauthState, ct);
        var url = googleOAuthService.GetAuthorizationUrl(redirectUri, state);
        logger.LogDebug("Started Gmail auth for {UserEmail} with state {State}", userEmail, state);

        return new StartGmailAuthResponseModel
        {
            Uri = url,
            State = state
        };
    }

    public async Task<string> HandleGmailCallbackAsync(HandleGmailCallbackRequestModel request, CancellationToken ct)
    {
        var requestEmail = request.UserEmail;
        logger.LogDebug("Handling Gmail callback for {UserEmail}", request.UserEmail);

        var storedState = await oauthStateRepository.GetState(request.State, ct);
        if (storedState == null || storedState.UserEmail != requestEmail)
        {
            logger.LogWarning("Invalid OAuth state for {UserEmail}", requestEmail);
            throw new OAuthException("Invalid authorization request");
        }

        await oauthStateRepository.DeleteState(request.State, ct);

        var tokenExchangeResult = await googleOAuthService.ExchangeCode(request.Code, request.RedirectUri, ct);

        var user = await userRepository.GetUserByEmail(requestEmail, ct);
        if (user == null)
            throw new InvalidOperationException($"User {requestEmail} not found");

        var gmailToken = new GmailToken
        {
            UserId = user.Id,
            AccessToken = tokenExchangeResult.AccessToken,
            RefreshToken = tokenExchangeResult.RefreshToken ?? throw new ArgumentException("Invalid refresh token"),
            ExpiresAt = tokenExchangeResult.ExpiresAt,
            GmailEmail = tokenExchangeResult.GmailEmail
        };

        await gmailTokenRepository.SaveToken(gmailToken, ct);

        logger.LogInformation("Successfully connected Gmail for {UserEmail} -> {GmailEmail}", requestEmail,
            tokenExchangeResult.GmailEmail);

        return tokenExchangeResult.GmailEmail;
    }



    public async Task<PersonageTokenModel> RegisterWithPassword(RegisterUserRequestModel request, CancellationToken ct)
    {
        UserValidator.ValidateUser(request.Email, request.Password, request.Name);

        var userByEmail = await userRepository.GetUserByEmail(request.Email, ct);
        if (userByEmail is not null)
            throw new CustomException(ErrorCode.UserAlreadyExists, $"User with email {request.Email} already exists");

        var hashedPassword = PasswordHasher.HashPassword(request.Password);

        var user = await userRepository.CreateUser(new CreateUserRequest
        {
            Email = request.Email,
            PasswordHash = hashedPassword,
            Name = request.Name,
        }, ct);
        var refreshToken = await tokenService.GenerateAndStoreRefreshToken(user.Id, ct);

        return new PersonageTokenModel
        {
            AccessToken = tokenService.GenerateAccessToken(user.Id),
            RefreshToken = refreshToken.Token
        };
    }

    public async Task<PersonageTokenModel> AuthByPassword(string email, string password, CancellationToken ct)
    {
        var user = await userRepository.GetUserByEmail(email, ct);
        if (user is null)
            throw new AuthenticationException(ErrorCode.InvalidCredentials, "Invalid user credentials");

        if (user.PasswordHash is null)
            throw new AuthenticationException(ErrorCode.PasswordNotSet, "Password for user not set");

        if (!PasswordHasher.VerifyPassword(password, user.PasswordHash))
            throw new AuthenticationException(ErrorCode.InvalidCredentials, "Invalid user credentials");

        var accessToken = tokenService.GenerateAccessToken(user.Id);
        var refreshToken = await tokenService.GenerateAndStoreRefreshToken(user.Id, ct);

        return new PersonageTokenModel
        {
            AccessToken = accessToken,
            RefreshToken = refreshToken.Token,
        };
    }

    public async Task InitiatePasswordReset(string email, string resetUrlBase, CancellationToken ct)
    {
        var user = await userRepository.GetUserByEmail(email, ct);
        if (user is null)
        {
            logger.LogInformation("Password reset requested for non-existent email: {Email}", email);
            return;
        }

        if (user.PasswordHash is null)
        {
            logger.LogWarning("Password reset requested for user without password: {Email}", email);
            return;
        }

        // Invalidate any existing reset tokens
        await passwordResetTokenRepository.InvalidateAllUserTokens(user.Id, ct);

        // Generate reset token
        var tokenBytes = new byte[32];
        using var rng = RandomNumberGenerator.Create();
        rng.GetBytes(tokenBytes);
        var token = Convert.ToBase64String(tokenBytes)
            .Replace('+', '-')
            .Replace('/', '_')
            .Replace("=", "");

        var expiresAt = DateTime.UtcNow.AddHours(1);
        await passwordResetTokenRepository.CreateToken(user.Id, token, expiresAt, ct);

        var resetLink = $"{resetUrlBase}?token={token}";
        await postboxService.SendPasswordResetEmail(email, resetLink, ct);

        logger.LogInformation("Password reset initiated for {Email}", email);
    }

    public async Task<PersonageTokenModel> ResetPassword(string token, string newPassword, CancellationToken ct)
    {
        var resetToken = await passwordResetTokenRepository.GetToken(token, ct);
        if (resetToken is null)
            throw new CustomException(ErrorCode.InvalidResetToken, "Invalid or expired reset token");

        var user = await userRepository.GetUserById(resetToken.UserId, ct);
        if (user is null)
            throw new AuthenticationException(ErrorCode.InvalidResetToken, "User not found");

        UserValidator.ValidatePassword(newPassword);

        var hashedPassword = PasswordHasher.HashPassword(newPassword);
        await userRepository.UpdatePassword(user.Id, hashedPassword, ct);

        await passwordResetTokenRepository.InvalidateToken(token, ct);
        var accessToken = tokenService.GenerateAccessToken(user.Id);
        var refreshToken = await tokenService.GenerateAndStoreRefreshToken(user.Id, ct);

        logger.LogInformation("Password reset completed for {Email}", user.Email);

        return new PersonageTokenModel
        {
            AccessToken = accessToken,
            RefreshToken = refreshToken.Token
        };
    }

    private static string GenerateState()
    {
        using var rng = RandomNumberGenerator.Create();
        var bytes = new byte[32];
        rng.GetBytes(bytes);
        return Convert.ToBase64String(bytes)
            .Replace('+', '-')
            .Replace('/', '_')
            .Replace("=", "");
    }
}
