using System.Threading.Tasks;
using AutoFixture;
using FluentAssertions;
using Grpc.Core;
using Microsoft.VisualStudio.TestTools.UnitTesting;
using Personage.Auth.Api.Grpc;

namespace Personage.Auth.Tests.GrpcServices;

[TestClass]
public class AuthGrpcServiceTests : TestClassBase
{
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
}