using Personage.Auth.Api.Contracts.User.Requests;
using RestEase;

namespace Personage.Auth.Tests.Api;

[BasePath("user")]
public interface IUserApi
{
    [Put]
    Task UpdateUserInfo(
        [Body] UpdateUserInfoRequest request,
        [Header("user_id")] string userId,
        CancellationToken ct
    );
}