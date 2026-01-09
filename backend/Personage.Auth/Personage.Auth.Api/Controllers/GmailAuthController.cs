using Microsoft.AspNetCore.Mvc;
using Personage.Auth.Contracts.Auth.Gmail.Requests;
using Personage.Auth.Domain.Exceptions;
using Personage.Auth.Domain.Interfaces;
using Personage.Auth.Domain.Models.Requests;

namespace Personage.Auth.Controllers;

[ApiController]
[Route("auth/gmail")]
public class GmailAuthController(
    IAuthService authService,
    ILogger<GmailAuthController> logger
) : ControllerBase
{
    [HttpPost("gmail/authorize")]
    public async Task<IActionResult> StartGmailAuth([FromBody] StartAuthRequest request, CancellationToken ct)
    {
        try
        {
            var (url, state) = await authService.StartGmailAuth(request.UserEmail, request.RedirectUri, ct);
            
            return Ok(new
            {
                AuthorizationUrl = url,
                State = state
            });
        }
        catch (Exception ex)
        {
            logger.LogError(ex, "Failed to start Gmail auth for {UserEmail}", request.UserEmail);
            return BadRequest(new { Error = "Failed to start authorization" });
        }
    }
    
    [HttpPost("gmail/callback")]
    public async Task<IActionResult> HandleGmailCallback([FromBody] AuthCallbackRequest request, CancellationToken ct)
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

            return Ok(new
            {
                Success = true,
                GmailEmail = gmailEmail
            });
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