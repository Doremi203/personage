using Microsoft.AspNetCore.Mvc;
using Personage.Auth.Api.Contracts.Auth.Gmail.Requests;
using Personage.Auth.Api.Contracts.Auth.Gmail.Responses;
using Personage.Auth.Api.Contracts.Common;
using Personage.Auth.Domain.Interfaces;
using Personage.Auth.Domain.Models.Auth.Gmail.Requests;
using StartAuthRequest = Personage.Auth.Api.Contracts.Auth.GoogleCalendar.Requests.StartAuthRequest;
using StartAuthResponse = Personage.Auth.Api.Contracts.Auth.GoogleCalendar.Responses.StartAuthResponse;

namespace Personage.Auth.Api.Controllers;

[ApiController]
[Route("auth/google-calendar")]
public class GoogleCalendarAuthController(
    IAuthService authService
) : ControllerBase
{
    [HttpPost("authorize")]
    [ProducesResponseType(typeof(StartAuthResponse), StatusCodes.Status200OK)]
    [ProducesResponseType(typeof(ErrorResponse), StatusCodes.Status400BadRequest)]
    public async Task<ActionResult<StartAuthResponse>> StartGmailAuth([FromBody] StartAuthRequest request, CancellationToken ct)
    {
        var res = await authService.StartGoogleCalendarAuth(request.RedirectUri, ct);

        return new StartAuthResponse
        {
            AuthorizationUrl = res.Uri,
            State = res.State,
        };
    }
    
    [HttpPost("callback")]
    [ProducesResponseType(typeof(AuthCallbackResponse), StatusCodes.Status200OK)]
    [ProducesResponseType(typeof(ErrorResponse), StatusCodes.Status400BadRequest)]
    public async Task<ActionResult<AuthCallbackResponse>> HandleGoogleCalendarCallback([FromBody] AuthCallbackRequest request, CancellationToken ct)
    {
        var gmailEmail = await authService.HandleGoogleCalendarCallback(
            new HandleOAuthCallbackRequestModel
            {
                UserEmail = request.UserEmail,
                Code = request.Code,
                State = request.State,
                RedirectUri = request.RedirectUri
            }, ct);

        return new AuthCallbackResponse
        {
            GmailEmail = gmailEmail
        };
    } 
}