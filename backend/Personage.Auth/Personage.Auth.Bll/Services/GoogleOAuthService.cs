using System.Text.Json;
using System.Text.Json.Serialization;
using System.Web;
using Microsoft.Extensions.Logging;
using Microsoft.Extensions.Options;
using Personage.Auth.Domain.Configuration;
using Personage.Auth.Domain.Exceptions;
using Personage.Auth.Domain.Interfaces;
using Personage.Auth.Domain.Models.Common;
using Personage.Auth.Domain.Models.GoogleAuth;

namespace Personage.Auth.Bll.Services;

public class GoogleOAuthService(
    HttpClient httpClient,
    IOptions<OAuthSettings> settings,
    ILogger<GoogleOAuthService> logger
) : IGoogleOAuthService
{
    private const string OAuthAuthorizationUrlPrefix = "https://accounts.google.com/o/oauth2/auth";
    private const string OAuthTokenUrlPrefix = "https://oauth2.googleapis.com/token";

    public string GetAuthorizationUrl(
        string redirectUri,
        string state,
        GoogleServiceKind serviceKind
    )
    {
        var scopes = GetOAuthScopes(serviceKind);
        var queryParams = new Dictionary<string, string>
        {
            ["client_id"] = settings.Value.ClientId,
            ["redirect_uri"] = redirectUri,
            ["response_type"] = "code",
            ["scope"] = string.Join(" ", scopes),
            ["access_type"] = "offline",
            ["state"] = state,
            ["prompt"] = "consent"
        };
        
        var queryString = string.Join("&", 
            queryParams.Select(kvp => $"{kvp.Key}={HttpUtility.UrlEncode(kvp.Value)}"));
        
        return $"{OAuthAuthorizationUrlPrefix}?{queryString}";
    }
    
    public async Task<OAuthTokenModel> ExchangeCode(string code, string redirectUri, CancellationToken ct)
    {
        var requestBody = new Dictionary<string, string>
        {
            ["code"] = code,
            ["client_id"] = settings.Value.ClientId,
            ["client_secret"] = settings.Value.ClientSecret,
            ["redirect_uri"] = redirectUri,
            ["grant_type"] = "authorization_code"
        };
        
        var response = await httpClient.PostAsync(
            OAuthTokenUrlPrefix,
            new FormUrlEncodedContent(requestBody), ct);
        
        if (!response.IsSuccessStatusCode)
        {
            var error = await response.Content.ReadAsStringAsync(ct);
            logger.LogError("Failed to exchange code: {Error}", error);
            throw new InvalidOperationException($"Failed to exchange code: {error}");
        }
        
        var json = await response.Content.ReadAsStringAsync(ct);
        var tokenResponse = JsonSerializer.Deserialize<AuthorizationTokenResponse>(json);
        
        if (tokenResponse == null)
            throw new InvalidOperationException("Invalid token response");
        
        var email = await GetUserEmailAsync(tokenResponse.AccessToken);
        
        return new OAuthTokenModel{
            AccessToken = tokenResponse.AccessToken,
            RefreshToken = tokenResponse.RefreshToken,
            ExpiresAt = DateTime.UtcNow.AddSeconds(tokenResponse.ExpiresIn),
            GmailEmail = email
        };
    }

    public async Task<OAuthTokenModel> RefreshToken(string refreshToken, CancellationToken ct)
    {
        var refreshRequest = new Dictionary<string, string>
        {
            ["refresh_token"] = refreshToken,
            ["client_id"] = settings.Value.ClientId,
            ["client_secret"] = settings.Value.ClientSecret,
            ["grant_type"] = "refresh_token"
        };

        var response = await httpClient.PostAsync(
            OAuthTokenUrlPrefix,
            new FormUrlEncodedContent(refreshRequest),
            ct);

        if (!response.IsSuccessStatusCode)
        {
            var errorContent = await response.Content.ReadAsStringAsync(ct);
            logger.LogError("Token refresh failed: {Error}", errorContent);
            
            if (errorContent.Contains("invalid_grant"))
                throw new OAuthException("Token has been revoked. User needs to re-authenticate.");
            
            throw new OAuthException($"Token refresh failed: {response.StatusCode}");
        }

        var json = await response.Content.ReadAsStringAsync(ct);
        var tokenResponse = JsonSerializer.Deserialize<RefreshTokenResponse>(json);

        if (tokenResponse == null)
            throw new OAuthException("Invalid token response from Google");

        return new OAuthTokenModel
        {
            AccessToken = tokenResponse.AccessToken,
            RefreshToken = tokenResponse.RefreshToken ?? refreshToken,
            ExpiresAt = DateTime.UtcNow.AddSeconds(tokenResponse.ExpiresIn),
            GmailEmail = string.Empty
        };
    }

    private string[] GetOAuthScopes(GoogleServiceKind serviceKind)
    {
        return serviceKind switch
        {
            GoogleServiceKind.Unknown => throw new ArgumentException("Invalid Google Service Kind"),
            GoogleServiceKind.Gmail => settings.Value.Scopes,
            GoogleServiceKind.Calendar => settings.Value.GoogleCalendarScopes,
            _ => throw new ArgumentOutOfRangeException(nameof(serviceKind), serviceKind, null)
        };
    }
    
    private async Task<string> GetUserEmailAsync(string accessToken)
    {
        var request = new HttpRequestMessage(HttpMethod.Get, "https://www.googleapis.com/oauth2/v2/userinfo");
        request.Headers.Authorization = new System.Net.Http.Headers.AuthenticationHeaderValue("Bearer", accessToken);
        
        var response = await httpClient.SendAsync(request);
        response.EnsureSuccessStatusCode();
        
        var json = await response.Content.ReadAsStringAsync();
        var userInfo = JsonSerializer.Deserialize<UserInfoResponse>(json);
        
        return userInfo?.Email ?? throw new InvalidOperationException("Could not get user email");
    }
    
    private class AuthorizationTokenResponse
    {
        [JsonPropertyName("access_token")]
        public string AccessToken { get; init; } = null!;
        
        [JsonPropertyName("expires_in")]
        public int ExpiresIn { get; init; }
        
        [JsonPropertyName("refresh_token")] 
        public string RefreshToken { get; init; } = null!;
    }
    
    private class RefreshTokenResponse
    {
        [JsonPropertyName("access_token")]
        public string AccessToken { get; init; } = null!;
        
        [JsonPropertyName("expires_in")]
        public int ExpiresIn { get; init; }
        
        [JsonPropertyName("refresh_token")] 
        public string? RefreshToken { get; init; }
    }
    
    private class UserInfoResponse
    {
        [JsonPropertyName("email")]
        public string Email { get; init; } = null!;
    }
}