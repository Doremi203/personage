using Microsoft.AspNetCore.Mvc;
using Personage.Auth.Domain.Interfaces;

namespace Personage.Auth.Api.Controllers;

[ApiController]
[Route("auth/personage")]
public class PersonageAuthController(
    IAuthService authService,
    ILogger<GmailAuthController> logger
) : ControllerBase
{