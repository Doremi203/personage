using System;
using System.Threading.Tasks;
using FluentAssertions;
using Microsoft.VisualStudio.TestTools.UnitTesting;
using Personage.Auth.Api.Grpc;

namespace Personage.Auth.Tests.GrpcServices;

[TestClass]
public class TestGrpcServiceTests : TestClassBase
{
    [TestMethod]
    public async Task GrpcPing_ReturnsOk()
    {
        // Act
        var response = await TestGrpcClient.PingAsync(new PingRequest());

        // Assert
        response.Message.Should().Be("Pong");
        response.Moment.ToDateTime().Should().BeCloseTo(DateTime.UtcNow, TimeSpan.FromMilliseconds(100));
    }
}