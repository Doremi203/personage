using Microsoft.AspNetCore.Mvc;
using Personage.Auth.Api.Contracts.Auth.OAuth.Requests;
using Personage.Auth.Api.Contracts.Auth.OAuth.Responses;
using Personage.Auth.Api.Contracts.Common;
using Personage.Auth.Domain.Interfaces;
using Personage.Auth.Domain.Models.Auth.Gmail.Requests;

namespace Personage.Auth.Api.Controllers;

[ApiController]
[Route("auth/gmail")]
public class GmailAuthController(
    IAuthService authService
) : ControllerBase
{
    [HttpPost("authorize")]
    [ProducesResponseType(typeof(StartAuthResponse), StatusCodes.Status200OK)]
    [ProducesResponseType(typeof(ErrorResponse), StatusCodes.Status400BadRequest)]
    public async Task<ActionResult<StartAuthResponse>> StartGmailAuth([FromBody] StartAuthRequest request, CancellationToken ct)
    {
        var res = await authService.StartGmailAuth(request.UserEmail, request.RedirectUri, ct);

        return new StartAuthResponse
        {
            AuthorizationUrl = res.Uri,
            State = res.State,
        };
    }
    
    [HttpPost("callback")]
    [ProducesResponseType(typeof(AuthCallbackResponse), StatusCodes.Status200OK)]
    [ProducesResponseType(typeof(ErrorResponse), StatusCodes.Status400BadRequest)]
    public async Task<ActionResult<AuthCallbackResponse>> HandleGmailCallback([FromBody] AuthCallbackRequest request, CancellationToken ct)
    {
        var gmailEmail = await authService.HandleGmailCallback(
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