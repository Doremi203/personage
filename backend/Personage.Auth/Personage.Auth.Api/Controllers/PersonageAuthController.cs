using Microsoft.AspNetCore.Mvc;
using Personage.Auth.Api.Contracts.Auth.Personage.Requests;
using Personage.Auth.Api.Contracts.Auth.Personage.Responses;
using Personage.Auth.Api.Contracts.Common;
using Personage.Auth.Domain.Interfaces;
using Personage.Auth.Domain.Models.Auth.Requests;

namespace Personage.Auth.Api.Controllers;

[ApiController]
[Route("auth/personage")]
public class PersonageAuthController(IAuthService authService) : ControllerBase
{
    [HttpPost("register")]
    [ProducesResponseType(typeof(PersonageAuthResponse), StatusCodes.Status200OK)]
    [ProducesResponseType(typeof(ErrorResponse), StatusCodes.Status409Conflict)]
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
    [ProducesResponseType(typeof(PersonageAuthResponse), StatusCodes.Status200OK)]
    [ProducesResponseType(typeof(ErrorResponse), StatusCodes.Status401Unauthorized)]
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
    
    [HttpPost("refresh")]
    [ProducesResponseType(typeof(PersonageAuthResponse), StatusCodes.Status200OK)]
    [ProducesResponseType(typeof(ErrorResponse), StatusCodes.Status401Unauthorized)]
    public async Task<ActionResult<PersonageAuthResponse>> AuthByRefreshToken(
        [FromBody] AuthByRefreshTokenRequest request,
        CancellationToken ct
    )
    {
        var res = await authService.RefreshAccessToken(request.RefreshToken, ct);

        return new PersonageAuthResponse
        {
            AccessToken = res.AccessToken,
            RefreshToken = res.RefreshToken,
        };
    }
    
    [HttpPost("forgot-password")]
    [ProducesResponseType(StatusCodes.Status200OK)]
    public async Task<ActionResult> ForgotPassword(
        [FromBody] ForgotPasswordRequest request,
        CancellationToken ct)
    {
        await authService.InitiatePasswordReset(request.Email, request.ResetUrlBase, ct);
        return Ok(new { message = "If the email exists, a reset link has been sent" });
    }

    [HttpPost("reset-password")]
    [ProducesResponseType(typeof(PersonageAuthResponse), StatusCodes.Status200OK)]
    [ProducesResponseType(typeof(ErrorResponse), StatusCodes.Status400BadRequest)]
    public async Task<ActionResult<PersonageAuthResponse>> ResetPassword(
        [FromBody] ResetPasswordRequest request,
        CancellationToken ct)
    {
        var res = await authService.ResetPassword(request.Token, request.NewPassword, ct);
    
        return new PersonageAuthResponse
        {
            AccessToken = res.AccessToken,
            RefreshToken = res.RefreshToken
        };
    }
}