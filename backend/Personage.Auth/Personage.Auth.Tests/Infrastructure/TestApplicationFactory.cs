using System;
using System.Linq;
using System.Net.Http;
using Microsoft.AspNetCore.Hosting;
using Microsoft.AspNetCore.Mvc.Testing;
using Microsoft.AspNetCore.TestHost;
using Microsoft.Extensions.DependencyInjection;
using Moq;
using Personage.Auth.Api;
using Personage.Auth.Bll.Services;
using Personage.Auth.Domain.Interfaces;
using Personage.Auth.Tests.Infrastructure.Repositories;

namespace Personage.Auth.Tests.Infrastructure;

public sealed class TestApplicationFactory(Action<IServiceCollection> overrideServices) : WebApplicationFactory<Program>
{
    public Mock<HttpMessageHandler> HttpMessageHandlerMock { get; } = new();
    
    protected override void ConfigureWebHost(IWebHostBuilder builder)
    {
        builder.UseEnvironment("Testing");
        
        builder.ConfigureTestServices(services =>
        {
            var descriptor = services.FirstOrDefault(
                d => d.ServiceType == typeof(IHttpClientFactory) ||
                     (d.ServiceType.IsGenericType && 
                      d.ServiceType.GetGenericTypeDefinition() == typeof(IHttpClientFactory)));
            
            if (descriptor != null) services.Remove(descriptor);

            services.AddHttpClient<IGoogleOAuthService, GoogleOAuthService>()
                .ConfigurePrimaryHttpMessageHandler(() => HttpMessageHandlerMock.Object);
            
            services.AddScoped<TestCleaners>();
            services.AddScoped<TestOAuthStateRepository>();
            
            overrideServices.Invoke(services);
        });
    }
}