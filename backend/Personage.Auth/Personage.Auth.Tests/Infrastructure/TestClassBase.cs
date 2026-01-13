using System;
using System.Net.Http;
using System.Threading.Tasks;
using AutoFixture;
using Grpc.Net.Client;
using Microsoft.AspNetCore.Mvc.Testing;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.VisualStudio.TestTools.UnitTesting;
using Personage.Auth.Api;
using Personage.Auth.Api.Grpc;
using Personage.Auth.Tests.Api;
using RestEase;

namespace Personage.Auth.Tests.Infrastructure;

public abstract class TestClassBase : IDisposable
{
    protected WebApplicationFactory<Program> Factory { get; }
    private HttpClient HttpClient { get; }
    private GrpcChannel GrpcChannel { get; }
    protected Fixture Fixture { get; } = new();
    protected Cleaner Cleaner { get; }
    
    //REST API
    protected ITestApi TestApi { get; }
    protected IGmailAuthApi GmailAuthApi { get; }
    
    //gRPC API
    protected TestService.TestServiceClient TestGrpcClient { get; }
    protected AuthService.AuthServiceClient AuthGrpcClient { get; }

    protected TestClassBase()

    {
        Factory = new TestApplicationFactory(OverrideServices);
        HttpClient = Factory.CreateClient();
        TestApi = RestClient.For<ITestApi>(HttpClient);
        GmailAuthApi = RestClient.For<IGmailAuthApi>(HttpClient);
        
        GrpcChannel = GrpcChannel.ForAddress("http://localhost", new GrpcChannelOptions
        {
            HttpClient = HttpClient
        });
        
        TestGrpcClient = new TestService.TestServiceClient(GrpcChannel);
        AuthGrpcClient = new AuthService.AuthServiceClient(GrpcChannel);

        Cleaner = Factory.Services.GetRequiredService<Cleaner>();
    }
    
    protected virtual void OverrideServices(IServiceCollection services)
    {
        services.AddSingleton<Cleaner>();
    }
    
    public void Dispose()
    {
        HttpClient.Dispose();
        Factory.Dispose();
        GrpcChannel.Dispose();
        GC.SuppressFinalize(this);
    }
    
    [TestCleanup]
    public async Task Cleanup()
    {
        await Cleaner.CleanCreatedObjects();
    }
}