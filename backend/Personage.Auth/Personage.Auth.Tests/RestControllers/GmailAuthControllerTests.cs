using System;
using System.Linq;
using System.Net;
using System.Net.Http;
using System.Text.Json;
using System.Threading;
using System.Threading.Tasks;
using AutoFixture;
using FluentAssertions;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Options;
using Microsoft.VisualStudio.TestTools.UnitTesting;
using Moq;
using Personage.Auth.Api.Contracts.Auth.Gmail.Requests;
using Personage.Auth.DataAccess.Interfaces.Repositories;
using Personage.Auth.DataAccess.Models;
using Personage.Auth.Domain.Configuration;
using Personage.Auth.Tests.Infrastructure;
using Personage.Auth.Tests.Infrastructure.Repositories;
using RestEase;

namespace Personage.Auth.Tests.RestControllers;

[TestClass]
public class GmailAuthControllerTests : TestClassBase
{
    private TestOAuthStateRepository TestOAuthStateRepository { get; }
    private IOAuthStateRepository OAuthStateRepository { get; }
    private IUserRepository UserRepository { get; }
    private TestCleaners TestCleaners { get; }

    private Mock<IOptions<OAuthSettings>> OAuthSettingsMock { get; } = new();


    public GmailAuthControllerTests()
    {
        TestOAuthStateRepository = Factory.Services.GetRequiredService<TestOAuthStateRepository>();
        OAuthStateRepository = Factory.Services.GetRequiredService<IOAuthStateRepository>();
        UserRepository = Factory.Services.GetRequiredService<IUserRepository>();
        TestCleaners = Factory.Services.GetRequiredService<TestCleaners>();
    }

    protected override void OverrideServices(IServiceCollection services)
    {
        base.OverrideServices(services);

        services.AddScoped(_ => OAuthSettingsMock.Object);
    }

    [TestMethod]
    public async Task StartGmailAuth_ShouldBeOk()
    {
        //arrange
        var userEmail = Fixture.Create<string>();
        var redirectUri = Fixture.Create<string>();
        var user = await UserRepository.CreateUser(userEmail, CancellationToken.None);
        var clientId = Fixture.Create<string>();
        var clientSecret = Fixture.Create<string>();
        var scopes = Fixture.CreateMany<string>().ToArray();

        OAuthSettingsMock
            .Setup(f => f.Value)
            .Returns(new OAuthSettings
            {
                ClientId = clientId,
                ClientSecret = clientSecret,
                Scopes = scopes
            });

        //act
        var res = await GmailAuthApi.StartGmailAuth(new StartAuthRequest
        {
            UserEmail = userEmail,
            RedirectUri = redirectUri
        }, CancellationToken.None);

        //assert
        var oAuthStateByEmail = await TestOAuthStateRepository.GetOAuthStateByUserEmail(userEmail);
        Cleaner.AddCleanAction(async () =>
        {
            await TestCleaners.DeleteUser(user.Id);
            await TestCleaners.DeleteOAuthState(oAuthStateByEmail!.State);
        });

        res.State.Should().Be(oAuthStateByEmail!.State);
        res.AuthorizationUrl.Should().ContainAll(values:
        [
            redirectUri,
            clientId,
            "client_id",
            "redirect_uri",
            "response_type",
            "scope",
            "access_type",
            "state",
            "prompt",
            "https://accounts.google.com/o/oauth2/auth",
            ..scopes
        ]);
    }

    [TestMethod]
    public async Task HandleGmailCallback_StateNotFound_ShouldThrow()
    {
        //arrange
        var userEmailWithNoState = Fixture.Create<string>();

        //act
        var ex = await GmailAuthApi.Invoking(async c => await c.StartGmailAuth(
                new StartAuthRequest
                {
                    UserEmail = userEmailWithNoState,
                    RedirectUri = Fixture.Create<string>()
                }, CancellationToken.None))
            .Should()
            .ThrowAsync<ApiException>();

        //assert
        ex.Which.StatusCode.Should().Be(HttpStatusCode.BadRequest);
    }

    [TestMethod]
    public async Task HandleGmailCallback_ShouldBeOk()
    {
        //arrange
        var userEmail = Fixture.Create<string>();
        var user = await UserRepository.CreateUser(userEmail, CancellationToken.None);
        var state = Fixture.Create<string>();
        var redirectUri = Fixture.Create<string>();
        var accessToken = Fixture.Create<string>();
        var refreshToken = Fixture.Create<string>();
        var expiresIn = Fixture.Create<int>();
        var code = Fixture.Create<string>();

        var clientId = Fixture.Create<string>();
        var clientSecret = Fixture.Create<string>();

        OAuthSettingsMock
            .Setup(f => f.Value)
            .Returns(new OAuthSettings
            {
                ClientId = clientId,
                ClientSecret = clientSecret,
                Scopes = []
            });

        var tokenResponse = new
        {
            access_token = accessToken,
            refresh_token = refreshToken,
            expires_in = expiresIn
        };

        var userInfoResponse = new
        {
            email = userEmail
        };

        await OAuthStateRepository.SaveState(new OAuthState
        {
            State = state,
            UserEmail = userEmail,
            RedirectUri = redirectUri,
            CreatedAt = Fixture.Create<DateTime>(),
            ExpiresAt = DateTime.UtcNow.AddHours(Random.Shared.Next(3, 10))
        }, CancellationToken.None);

        Cleaner.AddCleanAction(async () =>
        {
            await TestCleaners.DeleteOAuthState(state);
            await TestCleaners.DeleteUser(user.Id);
        });

        var factory = (TestApplicationFactory)Factory;
        var handlerMock = factory.HttpMessageHandlerMock;

        handlerMock.SetupSendAsync(
            "https://oauth2.googleapis.com/token",
            HttpMethod.Post,
            JsonSerializer.Serialize(tokenResponse));

        handlerMock.SetupSendAsync(
            "https://www.googleapis.com/oauth2/v2/userinfo",
            HttpMethod.Get,
            JsonSerializer.Serialize(userInfoResponse));

        //act
        var res = await GmailAuthApi.HandleGmailCallback(
            new AuthCallbackRequest
            {
                UserEmail = userEmail,
                Code = code,
                State = state,
                RedirectUri = redirectUri,
            }, CancellationToken.None);

        //assert
        res.GmailEmail.Should().Be(userEmail);
        handlerMock.VerifySendAsync(
            "https://oauth2.googleapis.com/token",
            HttpMethod.Post,
            Times.Once(),
            content =>
                content.Contains($"code={code}") &&
                content.Contains($"client_id={clientId}") &&
                content.Contains($"redirect_uri={Uri.EscapeDataString(redirectUri)}"));
    }
}