using Personage.Auth.Domain.Models.Auth.Requests;
using Personage.Auth.Domain.Models.Common;

namespace Personage.Auth.Domain.Interfaces;

public interface IAuthService
{
    Task<StartGmailAuthResponseModel> StartGmailAuth(string userEmail, string redirectUri, CancellationToken ct);
    Task<string> HandleGmailCallbackAsync(HandleGmailCallbackRequestModel request, CancellationToken ct);
    Task<PersonageTokenModel> AuthByPassword(string email, string password, CancellationToken ct);
    Task<PersonageTokenModel> RegisterWithPassword(RegisterUserRequestModel request, CancellationToken ct);
    Task InitiatePasswordReset(string email, string resetUrlBase, CancellationToken ct);
    Task<PersonageTokenModel> ResetPassword(string token, string newPassword, CancellationToken ct);
}