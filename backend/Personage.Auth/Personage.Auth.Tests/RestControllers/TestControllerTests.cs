using System;
using System.Threading;
using System.Threading.Tasks;
using FluentAssertions;
using Microsoft.VisualStudio.TestTools.UnitTesting;
using Personage.Auth.Tests.Infrastructure;

namespace Personage.Auth.Tests.RestControllers;

[TestClass]
public class TestControllerTests : TestClassBase
{
    [TestMethod]
    public async Task Ping_ReturnsOk()
    {
        //Act
        var response = await TestApi.PingAsync(CancellationToken.None);

        //Assert
        response.Message.Should().Be("Pong");
        response.Moment.Should().BeCloseTo(DateTime.Now, TimeSpan.FromMilliseconds(100));
    }
}