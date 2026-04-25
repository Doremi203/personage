using Microsoft.AspNetCore.Mvc;
using Personage.Auth.Api.Contracts.Integrations;
using Personage.Auth.Api.Contracts.Integrations.Requests;
using Personage.Auth.Domain.Interfaces;
using Personage.Auth.Domain.Models.Common;

namespace Personage.Auth.Api.Controllers;

[ApiController]
[Route("integrations")]
public class IntegrationsController(IIntegrationsService integrationsService) : ControllerBase
{
    [HttpPost("revoke-access")]
    [ProducesResponseType(StatusCodes.Status200OK)]
    public async Task<ActionResult> RevokeAccess([FromBody] RevokeAccessRequest request, CancellationToken ct)
    {
        var serviceType = request.IntegrationType switch
        {
            IntegrationType.Gmail => ServiceTypeModel.Gmail,
            IntegrationType.GoogleCalendar => ServiceTypeModel.GoogleCalendar,
            IntegrationType.Telegram => ServiceTypeModel.Telegram,
            _ => ServiceTypeModel.Unknown
        };

        await integrationsService.RevokeAccess(serviceType, ct);
        return Ok();
    }
}