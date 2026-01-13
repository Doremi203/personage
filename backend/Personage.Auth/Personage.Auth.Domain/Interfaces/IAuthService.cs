using Personage.Auth.Domain.Models.Requests;
using Personage.Auth.Domain.Models.Responses;

namespace Personage.Auth.Domain.Interfaces;

public interface IAuthService
{
    Task<StartGmailAuthResponseModel> StartGmailAuth(string userEmail, string redirectUri, CancellationToken ct);
    Task<string> HandleGmailCallbackAsync(HandleGmailCallbackRequestModel request, CancellationToken ct);
    Task<GmailTokenModel> GetUserGmailToken(string userEmail, CancellationToken ct);
}