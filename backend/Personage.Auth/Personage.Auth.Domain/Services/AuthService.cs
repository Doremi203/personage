using Microsoft.Extensions.Logging;
using Personage.Auth.DataAccess.Interfaces.Repositories;
using Personage.Auth.DataAccess.Models;
using Personage.Auth.Domain.Exceptions;
using Personage.Auth.Domain.Interfaces;
using Personage.Auth.Domain.Models.Requests;

namespace Personage.Auth.Domain.Services;

public class AuthService(
    IUserRepository userRepository,
    IGmailTokenRepository gmailTokenRepository,
    IOAuthStateRepository oauthStateRepository,
    IGoogleOAuthService googleOAuthService,
    ILogger<AuthService> logger
) : IAuthService
{
    public async Task<(string Url, string State)> StartGmailAuth(string userEmail, string redirectUri, CancellationToken ct)
    {
        if (await userRepository.GetUserByEmail(userEmail, ct) is null)
        {
            logger.LogWarning("User with email {Email} not found, creating new user", userEmail);
            await userRepository.CreateUser(userEmail, ct);
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
        
        return (url, state);
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
            RefreshToken = tokenExchangeResult.RefreshToken,
            ExpiresAt = tokenExchangeResult.ExpiresAt,
            GmailEmail = tokenExchangeResult.GmailEmail
        };
        
        await gmailTokenRepository.SaveToken(gmailToken, ct);
        
        logger.LogInformation("Successfully connected Gmail for {UserEmail} -> {GmailEmail}", requestEmail, tokenExchangeResult.GmailEmail);
        
        return tokenExchangeResult.GmailEmail;
    }
    
    private static string GenerateState()
    {
        using var rng = System.Security.Cryptography.RandomNumberGenerator.Create();
        var bytes = new byte[32];
        rng.GetBytes(bytes);
        return Convert.ToBase64String(bytes)
            .Replace('+', '-')
            .Replace('/', '_')
            .Replace("=", "");
    }
}