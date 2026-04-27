using AutoFixture;
using FluentAssertions;
using Grpc.Core;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.VisualStudio.TestTools.UnitTesting;
using Personage.Auth.Api.Grpc;
using Personage.Auth.DataAccess.Interfaces.Repositories;
using Personage.Auth.DataAccess.Models;
using Personage.Auth.Tests.Infrastructure;

namespace Personage.Auth.Tests.GrpcServices;

[TestClass]
public class AuthGrpcServiceTests : TestClassBase
{
    private IUserRepository UserRepository { get; }
    private IGmailTokenRepository TokenRepository { get; }
    private TestCleaners TestCleaners { get; }

    public AuthGrpcServiceTests()
    {
        UserRepository = Factory.Services.GetRequiredService<IUserRepository>();
        TokenRepository = Factory.Services.GetRequiredService<IGmailTokenRepository>();
        TestCleaners = Factory.Services.GetRequiredService<TestCleaners>();
    }

    [TestMethod]
    public async Task GetGmailTokens_NonExistentUser_ShouldThrowNotFound()
    {
        //arrange
        var nonExistentUserEmail = Fixture.Create<string>();

        //act
        var ex = await AuthGrpcClient.Invoking(async c => await c.GetGmailTokensAsync(new GetGmailTokensRequest
        {
            UserEmail = nonExistentUserEmail
        }))
            .Should()
            .ThrowAsync<RpcException>();
        //assert
        ex.Which.StatusCode.Should().Be(StatusCode.NotFound);
        ex.Which.Message.Should().Contain($"Token for user with email {nonExistentUserEmail} not found");
    }

    [TestMethod]
    public async Task GetGmailTokens_NonExpiredToken_ExistingUserAndHasToken_ShouldBeOk()
    {
        //arrange
        var userEmail = Fixture.Create<string>();
        var user = await UserRepository.CreateShortUser(userEmail, CancellationToken.None);
        var accessToken = Fixture.Create<string>();
        var refreshToken = Fixture.Create<string>();
        var gmailEmail = Fixture.Create<string>();
        var expiresAt = DateTime.UtcNow.AddMinutes(Random.Shared.Next(10, 50));

        Cleaner.AddCleanAction(async () =>
        {
            await TestCleaners.DeleteUser(user.Id);
        });

        await TokenRepository.SaveToken(new OAuthToken
        {
            UserId = user.Id,
            AccessToken = accessToken,
            RefreshToken = refreshToken,
            ExpiresAt = expiresAt,
            GmailEmail = gmailEmail
        }, CancellationToken.None);

        //act
        var res = await AuthGrpcClient.GetGmailTokensAsync(new GetGmailTokensRequest
        {
            UserEmail = userEmail
        });

        //assert
        res.AccessToken.Should().Be(accessToken);
        res.RefreshToken.Should().Be(refreshToken);
        res.ExpiresAt.ToDateTime().Should().BeCloseTo(expiresAt, TimeSpan.FromMilliseconds(100));
    }
}