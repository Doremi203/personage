using Microsoft.AspNetCore.Mvc;
using Personage.Auth.Api.Contracts.Auth.Gmail.Requests;
using Personage.Auth.Api.Contracts.Auth.Gmail.Responses;
using Personage.Auth.Domain.Exceptions;
using Personage.Auth.Domain.Interfaces;
using Personage.Auth.Domain.Models.Requests;

namespace Personage.Auth.Api.Controllers;

[ApiController]
[Route("auth/gmail")]
public class GmailAuthController(
    IAuthService authService,
    ILogger<GmailAuthController> logger
) : ControllerBase
{
    [HttpPost("authorize")]
    public async Task<ActionResult<StartAuthResponse>> StartGmailAuth([FromBody] StartAuthRequest request, CancellationToken ct)
    {
        try
        {
            var res = await authService.StartGmailAuth(request.UserEmail, request.RedirectUri, ct);

            return new StartAuthResponse
            {
                AuthorizationUrl = res.Uri,
                State = res.State,
            };
        }
        catch (Exception ex)
        {
            logger.LogError(ex, "Failed to start Gmail auth for {UserEmail}", request.UserEmail);
            return BadRequest(new { Error = "Failed to start authorization" });
        }
    }
    
    [HttpPost("callback")]
    public async Task<ActionResult<AuthCallbackResponse>> HandleGmailCallback([FromBody] AuthCallbackRequest request, CancellationToken ct)
    {
        try
        {
            var gmailEmail = await authService.HandleGmailCallbackAsync(
                new HandleGmailCallbackRequestModel
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
        catch (OAuthException ex)
        {
            logger.LogWarning(ex, "Unauthorized Gmail callback for {UserEmail}", request.UserEmail);
            return BadRequest(new { Error = "Invalid authorization request" });
        }
        catch (Exception ex)
        {
            logger.LogError(ex, "Failed to handle Gmail callback for {UserEmail}", request.UserEmail);
            return BadRequest(new { Error = "Failed to connect Gmail" });
        }
    } 
}