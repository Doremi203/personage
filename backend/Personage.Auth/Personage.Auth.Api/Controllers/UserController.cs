using Microsoft.AspNetCore.Authorization;
using Microsoft.AspNetCore.Mvc;
using Personage.Auth.Api.Contracts.Common;
using Personage.Auth.Api.Contracts.User.Integrations;
using Personage.Auth.Api.Contracts.User.Requests;
using Personage.Auth.Api.Contracts.User.Responses;
using Personage.Auth.Domain.Interfaces;

namespace Personage.Auth.Api.Controllers;

[ApiController]
[Route("user")]
public class UserController(IUserService userService) : ControllerBase
{
    [HttpGet]
    [Authorize]
    [ProducesResponseType(typeof(UserInfo), StatusCodes.Status200OK)]
    [ProducesResponseType(typeof(ErrorResponse), StatusCodes.Status404NotFound)]
    [ProducesResponseType(typeof(ErrorResponse), StatusCodes.Status401Unauthorized)]
    [ProducesResponseType(typeof(ErrorResponse), StatusCodes.Status403Forbidden)]
    public async Task<ActionResult<UserInfo>> GetUserInfo(CancellationToken ct)
    {
        var user = await userService.GetUserInfo(ct);

        return new UserInfo
        {
            Email = user.Email,
            Name = user.Name,
            GmailIntegration = new GmailIntegration
            {
                Enabled = user.GmailIntegration.Enabled,
                Gmail = user.GmailIntegration.Gmail
            },
            TelegramIntegration = new TelegramIntegration
            {
                Enabled = user.TelegramIntegration.Enabled
            },
            GoogleCalendarIntegration = new GoogleCalendarIntegration
            {
                Enabled = user.GoogleCalendarIntegration.Enabled,
                Gmail = user.GoogleCalendarIntegration.Gmail
            }
        };
    }
    
    [HttpPut]
    [Authorize]
    [ProducesResponseType(StatusCodes.Status200OK)]
    [ProducesResponseType(StatusCodes.Status400BadRequest)]
    public async Task<ActionResult> UpdateUserInfo(
        [FromBody] UpdateUserInfoRequest request,
        CancellationToken ct)
    {
        await userService.UpdateUserInfo(request.Name, ct);
        return Ok();
    }
}