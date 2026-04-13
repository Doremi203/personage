using System.IdentityModel.Tokens.Jwt;
using System.Security.Claims;
using System.Security.Cryptography;
using Microsoft.Extensions.Logging;
using Microsoft.Extensions.Options;
using Microsoft.IdentityModel.Tokens;
using Personage.Auth.Bll.Helpers;
using Personage.Auth.Bll.Helpers.Validation;
using Personage.Auth.DataAccess.Interfaces.Repositories;
using Personage.Auth.DataAccess.Models;
using Personage.Auth.DataAccess.Models.Requests;
using Personage.Auth.Domain.Configuration;
using Personage.Auth.Domain.Exceptions;
using Personage.Auth.Domain.Exceptions.Base;
using Personage.Auth.Domain.Interfaces;
using Personage.Auth.Domain.Models.Auth;
using Personage.Auth.Domain.Models.Auth.Gmail.Requests;
using Personage.Auth.Domain.Models.Auth.Gmail.Responses;
using Personage.Auth.Domain.Models.Common;
using Personage.Auth.Domain.Models.GoogleAuth;

namespace Personage.Auth.Bll.Services;

public class AuthService(
    IUserRepository userRepository,
    IGmailTokenRepository gmailTokenRepository,
    IOAuthStateRepository oauthStateRepository,
    IRefreshTokenRepository refreshTokenRepository,
    IPasswordResetTokenRepository passwordResetTokenRepository,
    IGoogleOAuthService googleOAuthService,
    IPostboxService postboxService,
    IOptionsMonitor<JwtSettings> jwtSettings,
    IClaimValues claimValues,
    ILogger<AuthService> logger
) : IAuthService
{
    private SigningCredentials? _signingCredentials;

    private const int TokenExpirationThresholdMinutes = 5;
    private JwtSecurityTokenHandler JwtHandler { get; } = new();


    public async Task<StartOAuthResponseModel> StartGmailAuth(string userEmail, string redirectUri,
        CancellationToken ct)
    {
        UserValidator.ValidateEmail(userEmail);
        //get user id from claims here and use it to check user and assign tokens
        if (await userRepository.GetUserByEmail(userEmail, ct) is not { } user)
        {
            //TODO: update flow, do not allow for gmail linking without an active account
            logger.LogWarning("User with email {Email} not found, creating new user", userEmail);
            throw new NotFoundException(ErrorCode.UserNotFound, "User with specified email not found");
        }

        return await StartOAuthInternal(redirectUri, user, GoogleServiceKind.Gmail, ct);
    }

    public async Task<StartOAuthResponseModel> StartGoogleCalendarAuth(string redirectUri, CancellationToken ct)
    {
        var userId = claimValues.GetUserId();
        
        if (await userRepository.GetUserById(userId, ct) is not {} user)
            throw new NotFoundException(ErrorCode.UserNotFound, "Invalid account. Try to relogin");
        
        return await StartOAuthInternal(redirectUri, user, GoogleServiceKind.Calendar, ct);
    }
    

    public async Task<string> HandleGmailCallbackAsync(HandleOAuthCallbackRequestModel request, CancellationToken ct)
    {
        var (userId, tokenExchangeResult) = await HandleGmailCallbackInternal(request, ct);

        var gmailToken = new GmailToken
        {
            UserId = userId,
            AccessToken = tokenExchangeResult.AccessToken,
            RefreshToken = tokenExchangeResult.RefreshToken ?? throw new ArgumentException("Invalid refresh token"),
            ExpiresAt = tokenExchangeResult.ExpiresAt,
            GmailEmail = tokenExchangeResult.GmailEmail
        };
        await gmailTokenRepository.SaveToken(gmailToken, ct);

        logger.LogInformation("Successfully connected Gmail for {UserEmail} -> {GmailEmail}", request.UserEmail,
            tokenExchangeResult.GmailEmail);

        return tokenExchangeResult.GmailEmail;
    }    

    
    private async Task<(Guid UserId, OAuthTokenModel TokenExchangeResult)> HandleGmailCallbackInternal(HandleOAuthCallbackRequestModel request, CancellationToken ct)
    {
        var requestEmail = request.UserEmail;
        logger.LogDebug("Handling OAuth callback for {UserEmail}", request.UserEmail);

        var storedState = await oauthStateRepository.GetState(request.State, ct);
        if (storedState == null || storedState.UserEmail != requestEmail)
        {
            logger.LogWarning("Invalid OAuth state for {UserEmail}", requestEmail);
            throw new OAuthException("Invalid authorization request");
        }

        var user = await userRepository.GetUserByEmail(requestEmail, ct);
        if (user == null)
            throw new InvalidOperationException($"User {requestEmail} not found");
        
        await oauthStateRepository.DeleteState(request.State, ct);
        return (user.Id, await googleOAuthService.ExchangeCode(request.Code, request.RedirectUri, ct));
    }
    
    
    
    

    public async Task<OAuthTokenModel> GetUserGmailToken(string userEmail, CancellationToken ct)
    {
        var userToken = await gmailTokenRepository.GetTokenByUserEmail(userEmail, ct);
        if (userToken is null)
            throw new TokenNotFoundException($"Token for user with email {userEmail} not found");

        if (userToken.ExpiresAt >= DateTime.UtcNow.AddMinutes(TokenExpirationThresholdMinutes))
            return new OAuthTokenModel
            {
                AccessToken = userToken.AccessToken,
                RefreshToken = userToken.RefreshToken,
                ExpiresAt = userToken.ExpiresAt,
                GmailEmail = userToken.GmailEmail
            };

        try
        {
            var refreshedToken = await googleOAuthService.RefreshToken(userToken.RefreshToken, ct);
            await gmailTokenRepository.UpdateToken(
                userToken.Id,
                refreshedToken.AccessToken,
                refreshedToken.RefreshToken ?? userToken.RefreshToken,
                refreshedToken.ExpiresAt,
                ct
            );
        }
        catch (Exception ex)
        {
            logger.LogError(ex, "Failed to refresh Gmail token for user {UserEmail}", userEmail);
            throw new OAuthException($"Token refresh failed: {ex.Message}");
        }

        return new OAuthTokenModel
        {
            AccessToken = userToken.AccessToken,
            RefreshToken = userToken.RefreshToken,
            ExpiresAt = userToken.ExpiresAt,
            GmailEmail = userToken.GmailEmail
        };
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
        var refreshToken = await GenerateAndStoreRefreshToken(user.Id, ct);

        return new PersonageTokenModel
        {
            AccessToken = GenerateAccessToken(user),
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

        var accessToken = GenerateAccessToken(user);
        var refreshToken = await GenerateAndStoreRefreshToken(user.Id, ct);

        return new PersonageTokenModel
        {
            AccessToken = accessToken,
            RefreshToken = refreshToken.Token,
        };
    }

    public async Task<PersonageTokenModel> RefreshAccessToken(string refreshToken, CancellationToken ct)
    {
        var storedToken = await refreshTokenRepository.GetRefreshToken(refreshToken, ct);

        if (storedToken == null ||
            storedToken.ExpiresAt <= DateTime.UtcNow
           )
            throw new AuthenticationException(ErrorCode.InvalidRefreshToken, "Invalid refresh token");

        var user = await userRepository.GetUserById(storedToken.UserId, ct);
        if (user is null)
            throw new AuthenticationException(ErrorCode.InvalidRefreshToken, "Invalid refresh token");

        var newAccessToken = GenerateAccessToken(user);

        return new PersonageTokenModel
        {
            AccessToken = newAccessToken,
            RefreshToken = refreshToken
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
        var accessToken = GenerateAccessToken(user);
        var refreshToken = await GenerateAndStoreRefreshToken(user.Id, ct);

        logger.LogInformation("Password reset completed for {Email}", user.Email);

        return new PersonageTokenModel
        {
            AccessToken = accessToken,
            RefreshToken = refreshToken.Token
        };
    }

    private async Task<StartOAuthResponseModel> StartOAuthInternal(
        string redirectUri,
        User user,
        GoogleServiceKind serviceKind,
        CancellationToken ct)
    {
        var state = GenerateState();
        var oauthState = new OAuthState
        {
            State = state,
            UserEmail = user.Email,
            RedirectUri = redirectUri,
            ExpiresAt = DateTime.UtcNow.AddMinutes(10)
        };

        await oauthStateRepository.SaveState(oauthState, ct);
        var url = googleOAuthService.GetAuthorizationUrl(redirectUri, state, serviceKind);
        logger.LogDebug("Started OAuth flow for {UserEmail} with state {State} for service kind: {ServiceKind}", user.Email, state, serviceKind);

        return new StartOAuthResponseModel
        {
            Uri = url,
            State = state
        };
    }
    
    private string GenerateAccessToken(User user)
    {
        var claims = new Claim[]
        {
            new(ClaimNames.UserId, user.Id.ToString())
        };

        var expiresAt = DateTime.UtcNow.AddMinutes(jwtSettings.CurrentValue.AccessTokenTtlMinutes);
        return GenerateToken(expiresAt, claims);
    }

    private async Task<RefreshToken> GenerateAndStoreRefreshToken(Guid userId, CancellationToken ct)
    {
        var tokenBytes = new byte[64];
        using var rng = RandomNumberGenerator.Create();
        rng.GetBytes(tokenBytes);
        var tokenString = Convert.ToBase64String(tokenBytes)
            .Replace('+', '-')
            .Replace('/', '_')
            .Replace("=", "");

        var createTokenRequest = new CreateRefreshTokenRequest
        {
            Token = tokenString,
            UserId = userId,
            ExpiresAt = DateTime.UtcNow.AddHours(jwtSettings.CurrentValue.RefreshTokenTtlHours)
        };

        return await refreshTokenRepository.CreateRefreshToken(createTokenRequest, ct);
    }

    private string GenerateToken(
        DateTime expiresAt,
        IEnumerable<Claim> claims
    )
    {
        var payload = new JwtPayload(
            issuer: jwtSettings.CurrentValue.Issuer,
            audience: jwtSettings.CurrentValue.Audience,
            claims: claims,
            notBefore: null,
            expires: expiresAt,
            issuedAt: DateTime.UtcNow);

        _signingCredentials ??= GetSigningCredentials();

        var token = new JwtSecurityToken(new JwtHeader(_signingCredentials), payload);
        return JwtHandler.WriteToken(token);
    }

    /// <summary>
    /// Creates RSA signing credentials from the PEM key provided via configuration.
    /// The PEM value is resolved from Lockbox by the configuration provider at startup.
    /// </summary>
    private SigningCredentials GetSigningCredentials()
    {
        var pem = jwtSettings.CurrentValue.PrivateKeyPem;
        if (string.IsNullOrWhiteSpace(pem))
            throw new InvalidOperationException(
                "JwtSettings:PrivateKeyPem is not configured. " +
                "In production, use the secret:{id}:{version}:{key} format.");

        return new SigningCredentials(CreateRsaSecurityKeyFromPem(pem), SecurityAlgorithms.RsaSha256);
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

    private static RsaSecurityKey CreateRsaSecurityKeyFromPem(string pem)
    {
        var rsa = RSA.Create();
        rsa.ImportFromPem(pem);
        return new RsaSecurityKey(rsa);
    }
}
