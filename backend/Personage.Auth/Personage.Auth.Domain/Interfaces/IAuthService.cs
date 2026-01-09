using Personage.Auth.Domain.Models.Requests;

namespace Personage.Auth.Domain.Interfaces;

public interface IAuthService
{
    Task<(string Url, string State)> StartGmailAuth(string userEmail, string redirectUri, CancellationToken ct);
    Task<string> HandleGmailCallbackAsync(HandleGmailCallbackRequestModel request, CancellationToken ct);
}