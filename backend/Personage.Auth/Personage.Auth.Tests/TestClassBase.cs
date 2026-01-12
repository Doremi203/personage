using System;
using System.Net.Http;
using AutoFixture;
using Grpc.Net.Client;
using Microsoft.AspNetCore.Mvc.Testing;
using Microsoft.VisualStudio.TestTools.UnitTesting;
using Personage.Auth.Api;
using Personage.Auth.Api.Grpc;
using Personage.Auth.Tests.Api;
using RestEase;

namespace Personage.Auth.Tests;

public abstract class TestClassBase : IDisposable
{
    private WebApplicationFactory<Program> Factory { get; set; } = null!;
    private HttpClient HttpClient { get; set; } = null!;
    private GrpcChannel GrpcChannel { get; set; } = null!;
    protected Fixture Fixture { get; } = new();
    
    //REST API
    protected ITestApi TestApi { get; private set; } = null!;
    
    //gRPC API
    protected TestService.TestServiceClient TestGrpcClient { get; private set; } = null!;
    protected AuthService.AuthServiceClient AuthGrpcClient { get; private set; } = null!;

    [TestInitialize]
    public virtual void TestInitialize()
    {
        Factory = new WebApplicationFactory<Program>();
        HttpClient = Factory.CreateClient();
        TestApi = RestClient.For<ITestApi>(HttpClient);
        
        GrpcChannel = GrpcChannel.ForAddress("http://localhost", new GrpcChannelOptions
        {
            HttpClient = HttpClient
        });
        
        TestGrpcClient = new TestService.TestServiceClient(GrpcChannel);
        AuthGrpcClient = new AuthService.AuthServiceClient(GrpcChannel);
    }
    
    public void Dispose()
    {
        HttpClient.Dispose();
        Factory.Dispose();
        GrpcChannel.Dispose();
        GC.SuppressFinalize(this);
    }
}