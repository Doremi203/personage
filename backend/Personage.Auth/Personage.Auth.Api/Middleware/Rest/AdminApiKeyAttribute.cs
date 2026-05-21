using Microsoft.AspNetCore.Mvc;
using Microsoft.AspNetCore.Mvc.Filters;
using Microsoft.Extensions.Options;
using Personage.Auth.Domain.Configuration;

namespace Personage.Auth.Api.Middleware.Rest;

[AttributeUsage(AttributeTargets.Class | AttributeTargets.Method)]
public class AdminApiKeyAttribute : Attribute, IAsyncAuthorizationFilter
{
    private const string HeaderName = "X-Admin-Key";

    public Task OnAuthorizationAsync(AuthorizationFilterContext context)
    {
        var settings = context.HttpContext.RequestServices
            .GetRequiredService<IOptions<AdminSettings>>().Value;

        if (string.IsNullOrEmpty(settings.ApiKey))
        {
            context.Result = new UnauthorizedResult();
            return Task.CompletedTask;
        }

        if (!context.HttpContext.Request.Headers.TryGetValue(HeaderName, out var provided)
            || provided.ToString() != settings.ApiKey)
        {
            context.Result = new UnauthorizedResult();
        }

        return Task.CompletedTask;
    }
}
