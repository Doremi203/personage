using Microsoft.AspNetCore.Mvc;

namespace Personage.Auth.Api.Controllers;

[ApiController]
public class InfrastructureController : ControllerBase
{
    [HttpGet("health")]
    public ActionResult Health()
    {
        return Ok();
    }

    [HttpGet("liveness")]
    public ActionResult Liveness()
    {
        return Ok();
    }
}