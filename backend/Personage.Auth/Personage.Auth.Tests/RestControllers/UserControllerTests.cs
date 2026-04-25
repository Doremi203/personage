using System.Security.Claims;
using AutoFixture;
using FluentAssertions;
using Microsoft.AspNetCore.Http;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.VisualStudio.TestTools.UnitTesting;
using Moq;
using Personage.Auth.Api.Contracts.User.Requests;
using Personage.Auth.DataAccess.Interfaces.Repositories;
using Personage.Auth.DataAccess.Models.Requests;
using Personage.Auth.Tests.Infrastructure;

namespace Personage.Auth.Tests.RestControllers;

[TestClass]
public class UserControllerTests : TestClassBase
{
    private IUserRepository UserRepository { get; }
    private TestCleaners TestCleaners { get; }
    private Mock<IHttpContextAccessor> HttpContextAccessorMock { get; } = new();

    public UserControllerTests()
    {
        UserRepository = Factory.Services.GetRequiredService<IUserRepository>();
        TestCleaners = Factory.Services.GetRequiredService<TestCleaners>();
    }

    protected override void OverrideServices(IServiceCollection services)
    {
        base.OverrideServices(services);
        
        services.AddScoped(_ => HttpContextAccessorMock.Object);
    }

    [TestMethod]
    public async Task UpdateUser_ShouldUpdateUserName()
    {
        //arrange
        var email = Fixture.Create<string>();
        var passwordHash = Fixture.Create<string>();
        var initialName = Fixture.Create<string>();
        var nameToBeSet = Fixture.Create<string>();
        
        var user = await UserRepository.CreateUser(new CreateUserRequest
        {
            Email = email,
            PasswordHash = passwordHash,
            Name = initialName
        }, CancellationToken.None);
        Cleaner.AddCleanAction(async () =>
        {
            await TestCleaners.DeleteUser(user.Id);
        });
        
        InitializeClientWithAuth(user.Id);

        HttpContextAccessorMock.SetupGet(x => x.HttpContext!.User.Claims)
            .Returns([
                new Claim("user_id", user.Id.ToString())
            ]);
        
        //act
        var userBeforeUpdate = await UserRepository.GetUserById(user.Id, CancellationToken.None);
        await UserApi.UpdateUserInfo(new UpdateUserInfoRequest
            {
                Name = nameToBeSet,
            },
            CancellationToken.None);
        var userAfterUpdate = await UserRepository.GetUserById(user.Id, CancellationToken.None);
        
        //assert
        userBeforeUpdate!.Email.Should().Be(email);
        userBeforeUpdate.Name.Should().Be(initialName);
        userBeforeUpdate.PasswordHash.Should().Be(passwordHash);
        
        userAfterUpdate!.Email.Should().Be(email);
        userAfterUpdate.Name.Should().Be(nameToBeSet);
        userAfterUpdate.PasswordHash.Should().Be(passwordHash);
    }
}