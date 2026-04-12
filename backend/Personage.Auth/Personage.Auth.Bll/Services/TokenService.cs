using System.IdentityModel.Tokens.Jwt;
using System.Security.Claims;
using System.Security.Cryptography;
using Microsoft.Extensions.Logging;
using Microsoft.Extensions.Options;
using Microsoft.IdentityModel.Tokens;
using Personage.Auth.DataAccess.Interfaces.Repositories;
using Personage.Auth.DataAccess.Models.Requests;
using Personage.Auth.Domain.Configuration;
using Personage.Auth.Domain.Exceptions;
using Personage.Auth.Domain.Exceptions.Base;
using Personage.Auth.Domain.Interfaces;
using Personage.Auth.Domain.Models.Auth;
using Personage.Auth.Domain.Models.Common;

namespace Personage.Auth.Bll.Services;

public class TokenService(
    IUserRepository userRepository,
    IGmailTokenRepository gmailTokenRepository,
    IRefreshTokenRepository refreshTokenRepository,
    IGoogleOAuthService googleOAuthService,
    IOptionsMonitor<JwtSettings> jwtSettings,
    ILogger<TokenService> logger
) : ITokenService
{
    private JwtSecurityTokenHandler JwtHandler { get; } = new();
    private SigningCredentials? _signingCredentials;
    private const int TokenExpirationThresholdMinutes = 5;

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

        var newAccessToken = GenerateAccessToken(user.Id);

        return new PersonageTokenModel
        {
            AccessToken = newAccessToken,
            RefreshToken = refreshToken
        };
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
    
    public bool VerifyToken(string token)
    {
        try
        {
            var validationParameters = GetTokenValidationParameters();
            JwtHandler.ValidateToken(token, validationParameters, out var validatedToken);
            return validatedToken is JwtSecurityToken;
        }
        catch (SecurityTokenInvalidSignatureException)
        {
            return false;
        }
        catch (Exception ex)
        {
            logger.LogError(ex, "Unexpected error during token verification");
            return false;
        }
    }

    private TokenValidationParameters GetTokenValidationParameters()
    {
        return new TokenValidationParameters
        {
            ValidateIssuerSigningKey = true,
            IssuerSigningKey = GetPublicSecurityKey(),
        };
    }

    public string GenerateAccessToken(Guid userId)
    {
        var claims = new Claim[]
        {
            new(ClaimNames.UserId, userId.ToString()),
        };

        var expiresAt = DateTime.UtcNow.AddMinutes(jwtSettings.CurrentValue.AccessTokenTtlMinutes);
        return GenerateToken(expiresAt, claims);
    }

    public async Task<RefreshTokenModel> GenerateAndStoreRefreshToken(Guid userId, CancellationToken ct)
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

        var dbToken = await refreshTokenRepository.CreateRefreshToken(createTokenRequest, ct);
        return new RefreshTokenModel
        {
            Id = dbToken.Id,
            Token = dbToken.Token,
            UserId = dbToken.UserId,
            ExpiresAt = dbToken.ExpiresAt,
            CreatedAt = dbToken.CreatedAt
        };
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
    

    private static RsaSecurityKey CreateRsaSecurityKeyFromPem(string pem)
    {
        var rsa = RSA.Create();
        rsa.ImportFromPem(pem);
        return new RsaSecurityKey(rsa);
    }

    private SecurityKey GetPublicSecurityKey()
    {
        var pem = jwtSettings.CurrentValue.PrivateKeyPem;
        if (string.IsNullOrWhiteSpace(pem))
            throw new InvalidOperationException("JwtSettings:PrivateKeyPem is not configured.");

        var rsa = RSA.Create();
        rsa.ImportFromPem(pem);

        var publicKeyRsa = RSA.Create();
        publicKeyRsa.ImportRSAPublicKey(rsa.ExportRSAPublicKey(), out _);

        return new RsaSecurityKey(publicKeyRsa);
    }
}