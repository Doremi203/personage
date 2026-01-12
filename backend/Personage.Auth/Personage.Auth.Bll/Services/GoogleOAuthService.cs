using System.Text.Json;
using System.Text.Json.Serialization;
using System.Web;
using Microsoft.Extensions.Logging;
using Microsoft.Extensions.Options;
using Personage.Auth.Domain.Configuration;
using Personage.Auth.Domain.Interfaces;
using Personage.Auth.Domain.Models.Responses;

namespace Personage.Auth.Bll.Services;

public class GoogleOAuthService(
    HttpClient httpClient,
    IOptions<OAuthSettings> settings,
    ILogger<GoogleOAuthService> logger
) : IGoogleOAuthService
{
    public string GetAuthorizationUrl(string redirectUri, string state)
    {
        var queryParams = new Dictionary<string, string>
        {
            ["client_id"] = settings.Value.ClientId,
            ["redirect_uri"] = redirectUri,
            ["response_type"] = "code",
            ["scope"] = string.Join(" ", settings.Value.Scopes),
            ["access_type"] = "offline",
            ["state"] = state,
            ["prompt"] = "consent"
        };
        
        var queryString = string.Join("&", 
            queryParams.Select(kvp => $"{kvp.Key}={HttpUtility.UrlEncode(kvp.Value)}"));
        
        return $"https://accounts.google.com/o/oauth2/auth?{queryString}";
    }
    
    public async Task<GmailTokenModel> ExchangeCode(string code, string redirectUri, CancellationToken ct)
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
            "https://oauth2.googleapis.com/token",
            new FormUrlEncodedContent(requestBody), ct);
        
        if (!response.IsSuccessStatusCode)
        {
            var error = await response.Content.ReadAsStringAsync(ct);
            logger.LogError("Failed to exchange code: {Error}", error);
            throw new InvalidOperationException($"Failed to exchange code: {error}");
        }
        
        var json = await response.Content.ReadAsStringAsync(ct);
        var tokenResponse = JsonSerializer.Deserialize<TokenResponse>(json);
        
        if (tokenResponse == null)
            throw new InvalidOperationException("Invalid token response");
        
        var email = await GetUserEmailAsync(tokenResponse.AccessToken);
        
        return new GmailTokenModel{
            AccessToken = tokenResponse.AccessToken,
            RefreshToken = tokenResponse.RefreshToken,
            ExpiresAt = DateTime.UtcNow.AddSeconds(tokenResponse.ExpiresIn),
            GmailEmail = email
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
    
    private class TokenResponse
    {
        [JsonPropertyName("access_token")]
        public string AccessToken { get; init; } = null!;
        
        [JsonPropertyName("refresh_token")]
        public string RefreshToken { get; init; } = null!;
        
        [JsonPropertyName("expires_in")]
        public int ExpiresIn { get; init; }
    }
    
    private class UserInfoResponse
    {
        [JsonPropertyName("email")]
        public string Email { get; init; } = null!;
    }
}