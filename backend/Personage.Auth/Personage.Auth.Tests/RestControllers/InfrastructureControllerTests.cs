using System.Threading.Tasks;
using FluentAssertions;
using Microsoft.VisualStudio.TestTools.UnitTesting;
using Personage.Auth.Tests.Infrastructure;

namespace Personage.Auth.Tests.RestControllers;

[TestClass]
public class InfrastructureControllerTests : TestClassBase
{
    [TestMethod]
    public async Task Health_ShouldBeOk()
    {
        //act + assert
        await InfrastructureApi
            .Invoking(async c => await c.Health())
            .Should()
            .NotThrowAsync();
    }

    [TestMethod]
    public async Task Liveness_ShouldBeOk()
    {
        //act + assert
        await InfrastructureApi
            .Invoking(async c => await c.Liveness())
            .Should()
            .NotThrowAsync();
    }
}