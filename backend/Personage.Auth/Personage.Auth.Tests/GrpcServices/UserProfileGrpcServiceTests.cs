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
public class UserProfileGrpcServiceTests : TestClassBase
{
    private IUserRepository UserRepository { get; }
    private IGmailTokenRepository TokenRepository { get; }
    private TestCleaners TestCleaners { get; }

    public UserProfileGrpcServiceTests()
    {
        UserRepository = Factory.Services.GetRequiredService<IUserRepository>();
        TokenRepository = Factory.Services.GetRequiredService<IGmailTokenRepository>();
        TestCleaners = Factory.Services.GetRequiredService<TestCleaners>();
    }

    [TestMethod]
    public async Task GetUserProfile_ExistingUser_ReturnsProfile()
    {
        //arrange
        var email = Fixture.Create<string>();
        var user = await UserRepository.CreateShortUser(email, CancellationToken.None);

        Cleaner.AddCleanAction(async () =>
        {
            await TestCleaners.DeleteUser(user.Id);
        });

        //act
        var res = await AuthGrpcClient.GetUserProfileAsync(new GetUserProfileRequest
        {
            UserId = user.Id.ToString()
        });

        //assert
        res.UserId.Should().Be(user.Id.ToString());
        res.Email.Should().Be(email);
        res.Name.Should().Be(user.Name ?? string.Empty);
        res.CreatedAt.Should().NotBeNull();
        res.ConnectedEmails.Should().BeEmpty();
    }

    [TestMethod]
    public async Task GetUserProfile_UserWithConnectedGmail_ReturnsConnectedEmails()
    {
        //arrange
        var email = Fixture.Create<string>();
        var gmailEmail = Fixture.Create<string>();
        var user = await UserRepository.CreateShortUser(email, CancellationToken.None);

        Cleaner.AddCleanAction(async () =>
        {
            await TestCleaners.DeleteUser(user.Id);
        });

        await TokenRepository.SaveToken(new OAuthToken
        {
            UserId = user.Id,
            AccessToken = Fixture.Create<string>(),
            RefreshToken = Fixture.Create<string>(),
            ExpiresAt = DateTime.UtcNow.AddHours(1),
            GmailEmail = gmailEmail
        }, CancellationToken.None);

        //act
        var res = await AuthGrpcClient.GetUserProfileAsync(new GetUserProfileRequest
        {
            UserId = user.Id.ToString()
        });

        //assert
        res.Email.Should().Be(email);
        res.ConnectedEmails.Should().ContainSingle().Which.Should().Be(gmailEmail);
    }

    [TestMethod]
    public async Task GetUserProfile_GmailEmailEqualsAuthEmail_OmitsFromConnectedEmails()
    {
        //arrange
        var email = Fixture.Create<string>();
        var user = await UserRepository.CreateShortUser(email, CancellationToken.None);

        Cleaner.AddCleanAction(async () =>
        {
            await TestCleaners.DeleteUser(user.Id);
        });

        await TokenRepository.SaveToken(new OAuthToken
        {
            UserId = user.Id,
            AccessToken = Fixture.Create<string>(),
            RefreshToken = Fixture.Create<string>(),
            ExpiresAt = DateTime.UtcNow.AddHours(1),
            GmailEmail = email.ToUpperInvariant()
        }, CancellationToken.None);

        //act
        var res = await AuthGrpcClient.GetUserProfileAsync(new GetUserProfileRequest
        {
            UserId = user.Id.ToString()
        });

        //assert
        res.Email.Should().Be(email);
        res.ConnectedEmails.Should().BeEmpty();
    }

    [TestMethod]
    public async Task GetUserProfile_NonExistentUser_ShouldThrowNotFound()
    {
        //arrange
        var nonExistentUserId = Guid.NewGuid();

        //act
        var ex = await AuthGrpcClient.Invoking(async c => await c.GetUserProfileAsync(new GetUserProfileRequest
        {
            UserId = nonExistentUserId.ToString()
        }))
            .Should()
            .ThrowAsync<RpcException>();

        //assert
        ex.Which.StatusCode.Should().Be(StatusCode.NotFound);
    }

    [TestMethod]
    public async Task GetUserProfile_InvalidUuid_ShouldThrowInvalidArgument()
    {
        //arrange
        var malformed = "not-a-uuid";

        //act
        var ex = await AuthGrpcClient.Invoking(async c => await c.GetUserProfileAsync(new GetUserProfileRequest
        {
            UserId = malformed
        }))
            .Should()
            .ThrowAsync<RpcException>();

        //assert
        ex.Which.StatusCode.Should().Be(StatusCode.InvalidArgument);
    }
}
