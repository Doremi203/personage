using System;
using Microsoft.AspNetCore.Hosting;
using Microsoft.AspNetCore.Mvc.Testing;
using Microsoft.AspNetCore.TestHost;
using Microsoft.Extensions.DependencyInjection;

using Personage.Auth.Api;

namespace Personage.Auth.Tests.Infrastructure;

public sealed class TestApplicationFactory(Action<IServiceCollection> overrideServices) : WebApplicationFactory<Program>
{
    protected override void ConfigureWebHost(IWebHostBuilder builder)
    {
        builder.UseEnvironment("Testing");
        
        builder.ConfigureTestServices(services =>
        {
            services.AddScoped<TestCleaners>();
            overrideServices.Invoke(services);
        });
    }
}