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
using Personage.Auth.Domain.Models.Auth.Requests;
using Personage.Auth.Domain.Models.Common;

namespace Personage.Auth.Bll.Services;

public class AuthService(
    IUserRepository userRepository,
    IGmailTokenRepository gmailTokenRepository,
    IOAuthStateRepository oauthStateRepository,
    IRefreshTokenRepository refreshTokenRepository,
    IGoogleOAuthService googleOAuthService,
    IOptionsMonitor<JwtSettings> jwtSettings,
    ILogger<AuthService> logger) : IAuthService
{
    private static readonly RsaSecurityKey DevRsa = new(RSA.Create(2048));

    private readonly SigningCredentials _signingCredentials = new(
        string.IsNullOrEmpty(jwtSettings.CurrentValue.PrivateKey)
            ? DevRsa
            : CreateRsaSecurityKeyFromPem(jwtSettings.CurrentValue.PrivateKey), SecurityAlgorithms.RsaSha256);


    private const int TokenExpirationThresholdMinutes = 5;
    private JwtSecurityTokenHandler JwtHandler { get; } = new();


    public async Task<StartGmailAuthResponseModel> StartGmailAuth(string userEmail, string redirectUri,
        CancellationToken ct)
    {
        //TODO: validate whether email is valid https://tracker.yandex.ru/PERSONAGE-59
        if (await userRepository.GetUserByEmail(userEmail, ct) is null)
        {
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

    public async Task<GmailTokenModel> GetUserGmailToken(string userEmail, CancellationToken ct)
    {
        var userToken = await gmailTokenRepository.GetTokenByUserEmail(userEmail, ct);
        if (userToken is null)
            throw new TokenNotFoundException($"Token for user with email {userEmail} not found");

        if (userToken.ExpiresAt >= DateTime.UtcNow.AddMinutes(TokenExpirationThresholdMinutes))
            return new GmailTokenModel
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

        return new GmailTokenModel
        {
            AccessToken = userToken.AccessToken,
            RefreshToken = userToken.RefreshToken,
            ExpiresAt = userToken.ExpiresAt,
            GmailEmail = userToken.GmailEmail
        };
    }
    
    public async Task<PersonageTokenModel> RegisterWithPassword(RegisterUserRequestModel request, CancellationToken ct)    {
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
            throw new AuthenticationException(ErrorCode.UserNotFound, "Invalid user credentials");

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
        if(user is null)
            throw new AuthenticationException(ErrorCode.InvalidRefreshToken, "Invalid refresh token");
        
        var newAccessToken = GenerateAccessToken(user);

        return new PersonageTokenModel
        {
            AccessToken = newAccessToken,
            RefreshToken = refreshToken
        };
    }

    private string GenerateAccessToken(User user)
    {
        var claims = new Claim[]
        {
            new(ClaimNames.Id, user.Id.ToString())
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

        var createTokenRequest = new CreateRefreshTokenRequest()
        {
            Token = tokenString,
            UserId = userId,
            ExpiresAt = DateTime.UtcNow.AddHours(jwtSettings.CurrentValue.RefreshTokenTtlHours)
        };

        return await refreshTokenRepository.CreateRefreshToken(createTokenRequest, ct);
    }

    private string GenerateToken(DateTime expiresAt, IEnumerable<Claim> claims)
    {
        var payload = new JwtPayload(
            issuer: jwtSettings.CurrentValue.Issuer,
            audience: jwtSettings.CurrentValue.Audience,
            claims: claims,
            notBefore: null,
            expires: expiresAt,
            issuedAt: DateTime.UtcNow);

        var token = new JwtSecurityToken(new JwtHeader(_signingCredentials), payload);
        return JwtHandler.WriteToken(token);
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