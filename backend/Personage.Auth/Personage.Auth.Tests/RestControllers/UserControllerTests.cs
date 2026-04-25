using AutoFixture;
using FluentAssertions;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.VisualStudio.TestTools.UnitTesting;
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

    public UserControllerTests()
    {
        UserRepository = Factory.Services.GetRequiredService<IUserRepository>();
        TestCleaners = Factory.Services.GetRequiredService<TestCleaners>();
    }

    [TestMethod]
    public async Task UpdateUser_ShouldUpdateUserName()
    {
        //arrange
        var email = Fixture.Create<string>();
        var passwordHash = Fixture.Create<string>();
        var initialName = Fixture.Create<string>();
        const string nameToBeSet = "New Valid Name";
        
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