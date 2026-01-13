using Microsoft.AspNetCore.Mvc;
using Personage.Auth.Api.Contracts.Test.Responses;

namespace Personage.Auth.Api.Controllers;

[ApiController]
[Route("test")]
public class TestController : ControllerBase
{
    [HttpGet("ping")]
    public ActionResult<PingResponse> Ping()
    {
        return Ok(new PingResponse
        {
            Message = "Pong",
            Moment = DateTime.UtcNow
        });
    }     
}