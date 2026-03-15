using System.Security.Claims;
using Microsoft.AspNetCore.Http;
using Personage.Auth.Domain.Exceptions.Base;
using Personage.Auth.Domain.Interfaces;
using Personage.Auth.Domain.Models.Auth;

namespace Personage.Auth.Bll.Services;

public class ClaimValues(IHttpContextAccessor httpContextAccessor) : IClaimValues
{
    private ClaimsPrincipal? User => httpContextAccessor.HttpContext?.User;

    public Guid GetUserId()
    {
        var userIdClaim = User?.FindFirst(ClaimNames.UserId)?.Value;
        if (string.IsNullOrEmpty(userIdClaim))
            throw new AuthenticationException(ErrorCode.InvalidClaims, "User ID not found in token");

        if (!Guid.TryParse(userIdClaim, out var userId))
            throw new AuthenticationException(ErrorCode.InvalidClaims, "Invalid user ID format in token");

        return userId;
    }
}