using Microsoft.AspNetCore.Mvc;
using Personage.Auth.Api.Contracts.Auth.Personage.Requests;
using Personage.Auth.Api.Contracts.Auth.Personage.Responses;
using Personage.Auth.Domain.Interfaces;
using Personage.Auth.Domain.Models.Auth.Requests;

namespace Personage.Auth.Api.Controllers;

//TODO: add global exception filter
[ApiController]
[Route("auth/personage")]
public class PersonageAuthController(IAuthService authService) : ControllerBase
{
    [HttpPost("register")]
    public async Task<ActionResult<PersonageAuthResponse>> Register(
        [FromBody] RegisterWithPasswordRequest request,
        CancellationToken ct
    )
    {
        var res = await authService.RegisterWithPassword(new RegisterUserRequestModel
        {
            Email = request.Email,
            Name = request.Name,
            Password = request.Password
        }, ct);

        return new PersonageAuthResponse
        {
            AccessToken = res.AccessToken,
            RefreshToken = res.RefreshToken
        };
    }

    [HttpPost("login/password")]
    public async Task<ActionResult<PersonageAuthResponse>> LoginWithPassword(
        [FromBody] LoginWithPasswordRequest request,
        CancellationToken ct
    )
    {
        var res = await authService.AuthByPassword(request.Email, request.Password, ct);

        return new PersonageAuthResponse
        {
            AccessToken = res.AccessToken,
            RefreshToken = res.RefreshToken,
        };
    }
}