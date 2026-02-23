using System.IdentityModel.Tokens.Jwt;
using System.Security.Claims;
using System.Security.Cryptography;
using Microsoft.Extensions.Logging;
using Microsoft.Extensions.Options;
using Microsoft.IdentityModel.Tokens;
using Personage.Auth.DataAccess.Interfaces.Repositories;
using Personage.Auth.DataAccess.Models;
using Personage.Auth.Domain.Configuration;
using Personage.Auth.Domain.Exceptions;
using Personage.Auth.Domain.Exceptions.Base;
using Personage.Auth.Domain.Interfaces;
using Personage.Auth.Domain.Models.Auth;
using Personage.Auth.Domain.Models.Auth.Requests;
using Personage.Auth.Domain.Models.Common;

namespace Personage.Auth.Bll.Services;

public class AuthService : IAuthService
{
    private static readonly RsaSecurityKey DevRsa = new(RSA.Create(2048));
    
    private readonly IUserRepository _userRepository;
    private readonly IGmailTokenRepository _gmailTokenRepository;
    private readonly IOAuthStateRepository _oauthStateRepository;
    private readonly IGoogleOAuthService _googleOAuthService;
    private readonly IOptionsMonitor<JwtSettings> _jwtSettings;
    private readonly ILogger<AuthService> _logger;
    private readonly SigningCredentials _signingCredentials;

    public AuthService(IUserRepository userRepository,
        IGmailTokenRepository gmailTokenRepository,
        IOAuthStateRepository oauthStateRepository,
        IGoogleOAuthService googleOAuthService,
        IOptionsMonitor<JwtSettings> jwtSettings,
        ILogger<AuthService> logger)
    {
        _userRepository = userRepository;
        _gmailTokenRepository = gmailTokenRepository;
        _oauthStateRepository = oauthStateRepository;
        _googleOAuthService = googleOAuthService;
        _jwtSettings = jwtSettings;
        _logger = logger;
        
        var privateKey = string.IsNullOrEmpty(_jwtSettings.CurrentValue.PrivateKey)
            ? DevRsa
            : CreateRsaSecurityKeyFromPem(_jwtSettings.CurrentValue.PrivateKey);
        _signingCredentials = new SigningCredentials(privateKey, SecurityAlgorithms.RsaSha256);
    }

    private const int TokenExpirationThresholdMinutes = 5;
    private JwtSecurityTokenHandler JwtHandler { get; } = new();
    

    public async Task<StartGmailAuthResponseModel> StartGmailAuth(string userEmail, string redirectUri, CancellationToken ct)
    {
        //TODO: validate whether email is valid https://tracker.yandex.ru/PERSONAGE-59
        if (await _userRepository.GetUserByEmail(userEmail, ct) is null)
        {
            _logger.LogWarning("User with email {Email} not found, creating new user", userEmail);
            await _userRepository.CreateUser(userEmail, ct);
        }

        var state = GenerateState();
        
        var oauthState = new OAuthState
        {
            State = state,
            UserEmail = userEmail,
            RedirectUri = redirectUri,
            ExpiresAt = DateTime.UtcNow.AddMinutes(10)
        };
        
        await _oauthStateRepository.SaveState(oauthState, ct);
        var url = _googleOAuthService.GetAuthorizationUrl(redirectUri, state);
        _logger.LogDebug("Started Gmail auth for {UserEmail} with state {State}", userEmail, state);
        
        return new StartGmailAuthResponseModel
        {
            Uri = url,
            State = state
        };
    }
    
    public async Task<string> HandleGmailCallbackAsync(HandleGmailCallbackRequestModel request, CancellationToken ct)
    {
        var requestEmail = request.UserEmail;
        _logger.LogDebug("Handling Gmail callback for {UserEmail}", request.UserEmail);
        
        var storedState = await _oauthStateRepository.GetState(request.State, ct);
        if (storedState == null || storedState.UserEmail != requestEmail)
        {
            _logger.LogWarning("Invalid OAuth state for {UserEmail}", requestEmail);
            throw new OAuthException("Invalid authorization request");
        }
        
        await _oauthStateRepository.DeleteState(request.State, ct);
        
        var tokenExchangeResult = await _googleOAuthService.ExchangeCode(request.Code, request.RedirectUri, ct);
        
        var user = await _userRepository.GetUserByEmail(requestEmail, ct);
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
        
        await _gmailTokenRepository.SaveToken(gmailToken, ct);
        
        _logger.LogInformation("Successfully connected Gmail for {UserEmail} -> {GmailEmail}", requestEmail, tokenExchangeResult.GmailEmail);
        
        return tokenExchangeResult.GmailEmail;
    }

    public async Task<GmailTokenModel> GetUserGmailToken(string userEmail, CancellationToken ct)
    {
        var userToken = await _gmailTokenRepository.GetTokenByUserEmail(userEmail, ct);
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
            var refreshedToken = await _googleOAuthService.RefreshToken(userToken.RefreshToken, ct);
            await _gmailTokenRepository.UpdateToken(
                userToken.Id,
                refreshedToken.AccessToken,
                refreshedToken.RefreshToken ?? userToken.RefreshToken,
                refreshedToken.ExpiresAt,
                ct
            );
        }
        catch (Exception ex)
        {
            _logger.LogError(ex, "Failed to refresh Gmail token for user {UserEmail}", userEmail);
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

    public async Task<PersonageTokenModel> AuthByPassword(string email, string password, CancellationToken ct)
    {
        var user = await _userRepository.GetUserByEmail(email, ct);
        if (user is null)
            throw new AuthenticationException(ErrorCode.UserNotFound, "Invalid user credentials");

        var accessToken = GenerateAccessToken(user);
        
        return new PersonageTokenModel
        {
            AccessToken = accessToken,
            RefreshToken = null
        }
    }

    private string GenerateAccessToken(User user)
    {
        var claims = new Claim[]
        {
            new(ClaimNames.Id, user.Id.ToString())
        };

        var expiresAt = DateTime.UtcNow.AddMinutes(_jwtSettings.CurrentValue.AccessTokenTtlMinutes);
        var payload = new JwtPayload(
            issuer: _jwtSettings.CurrentValue.Issuer,
            audience: _jwtSettings.CurrentValue.Audience,
            claims: claims,
            notBefore: null,
            expires: expiresAt,
            issuedAt: DateTime.UtcNow);
        
        var token = new JwtSecurityToken(new JwtHeader(_signingCredentials), payload);
        return JwtHandler.WriteToken(token);
    }
    
    protected internal async Task<string> GenerateRefreshToken(
        CancellationToken ct,
        long userId,
        TimeSpan? refreshTokenTTL,
        string device,
        string appName,
        IEnumerable<KeyValuePair<string, object>> claims)
    {
        var utcNow = DateTime.UtcNow;
        var expiredUtc = Truncate(utcNow + (refreshTokenTTL ?? _lmsAuthJwtHandler.GetRefreshExpiredTTL()));

        await _userDeviceRepository.CreateOrUpdateAsync(userId, device, ct);

        claims ??= Enumerable.Empty<KeyValuePair<string, object>>();
        claims = claims
            .Append(new KeyValuePair<string, object>(Names.UserId, userId))
            .Append(new KeyValuePair<string, object>(Names.Device, device))
            .Append(new KeyValuePair<string, object>(Names.AppName, appName))
            .Append(new KeyValuePair<string, object>(Names.IsV2Token, true));

        var refreshToken = _lmsAuthJwtHandler.WriteRefreshToken(claims, utcNow, expiredUtc);

        return refreshToken;
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