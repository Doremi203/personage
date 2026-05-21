using Microsoft.AspNetCore.Authorization;
using Microsoft.AspNetCore.Mvc;
using Personage.Auth.Api.Contracts.Common;
using Personage.Auth.Api.Contracts.TelegramChats;
using Personage.Auth.Domain.Interfaces;

namespace Personage.Auth.Api.Controllers;

[ApiController]
[Authorize]
[Route("")]
public class TelegramChatsController(
    ITelegramChatsService telegramChatsService,
    IClaimValues claimValues
) : ControllerBase
{
    [HttpPost("get-user-chats")]
    [ProducesResponseType(typeof(GetUserChatsResponse), StatusCodes.Status200OK)]
    [ProducesResponseType(typeof(ErrorResponse), StatusCodes.Status404NotFound)]
    [ProducesResponseType(typeof(ErrorResponse), StatusCodes.Status401Unauthorized)]
    public async Task<ActionResult<GetUserChatsResponse>> GetUserChats(CancellationToken ct)
    {
        var userId = claimValues.GetUserId();
        var chats = await telegramChatsService.GetUserChats(userId, ct);
        return new GetUserChatsResponse
        {
            Chats = chats
                .Select(c => new TelegramChatDto
                {
                    ChatId = c.ChatId,
                    ChatName = c.ChatName,
                    IsActive = c.IsActive,
                })
                .ToList()
        };
    }

    [HttpPut("update-user-chat")]
    [ProducesResponseType(StatusCodes.Status200OK)]
    [ProducesResponseType(typeof(ErrorResponse), StatusCodes.Status404NotFound)]
    [ProducesResponseType(typeof(ErrorResponse), StatusCodes.Status401Unauthorized)]
    public async Task<ActionResult> UpdateUserChat(
        [FromBody] UpdateUserChatRequest request,
        CancellationToken ct)
    {
        var userId = claimValues.GetUserId();
        await telegramChatsService.UpdateActiveStatus(userId, request.ChatId, request.IsActive, ct);
        return Ok();
    }
}
