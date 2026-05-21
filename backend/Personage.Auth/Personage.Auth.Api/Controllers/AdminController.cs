using Microsoft.AspNetCore.Mvc;
using Personage.Auth.Api.Middleware.Rest;
using Personage.Auth.DataAccess.Interfaces.Repositories;

namespace Personage.Auth.Api.Controllers;

[ApiController]
[Route("admin")]
[AdminApiKey]
public class AdminController(IUserRepository userRepository) : ControllerBase
{
    public class AdminUser
    {
        public Guid Id { get; init; }
        public string Email { get; init; } = null!;
        public string? Name { get; init; }
    }

    public class ListAdminUsersResponse
    {
        public IReadOnlyList<AdminUser> Users { get; init; } = [];
    }

    [HttpGet("users")]
    [ProducesResponseType(typeof(ListAdminUsersResponse), StatusCodes.Status200OK)]
    [ProducesResponseType(StatusCodes.Status401Unauthorized)]
    public async Task<ActionResult<ListAdminUsersResponse>> ListUsers(CancellationToken ct)
    {
        var users = await userRepository.GetAllUsers(ct);

        return new ListAdminUsersResponse
        {
            Users = users
                .Select(u => new AdminUser
                {
                    Id = u.Id,
                    Email = u.Email,
                    Name = u.Name
                })
                .ToArray()
        };
    }
}
