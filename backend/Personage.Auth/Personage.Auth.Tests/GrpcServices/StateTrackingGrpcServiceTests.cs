using System;
using System.Collections.Generic;
using System.Linq;
using System.Threading;
using System.Threading.Tasks;
using FluentAssertions;
using Google.Protobuf.WellKnownTypes;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.VisualStudio.TestTools.UnitTesting;
using Personage.Auth.Api.Grpc.State;
using Personage.Auth.DataAccess.Models;
using Personage.Auth.Tests.Infrastructure;
using Personage.Auth.Tests.Infrastructure.Repositories;
using ServiceType = Personage.Auth.Api.Grpc.Common.ServiceType;

namespace Personage.Auth.Tests.GrpcServices;

[TestClass]
public class StateTrackingGrpcServiceTests : TestClassBase
{
    private TestUserRepository TestUserRepository { get; }
    private TestCleaners TestCleaners { get; }

    public StateTrackingGrpcServiceTests()
    {
        TestCleaners = Factory.Services.GetRequiredService<TestCleaners>();
        TestUserRepository = Factory.Services.GetRequiredService<TestUserRepository>();
    }

    [TestMethod]
    public async Task GetUsersForProcessing_ShouldReturnUserWithTokenNotProcessedForSpecifiedTime()
    {
        //arrange
        const int notProcessedForMinutesThreshold = 15;
        var matchingTimestamp = DateTime.UtcNow
            .AddMinutes(-notProcessedForMinutesThreshold - Random.Shared.Next(5, 100));

        var nonMatchingTimestamp = DateTime.UtcNow
            .AddMinutes(-Random.Shared.Next(1, notProcessedForMinutesThreshold - 1));
        
        var userNotProcessed = await TestUserRepository.CreateUserWithToken(null);
        var userProcessedBeforeCutoff = await TestUserRepository.CreateUserWithToken(matchingTimestamp);
        var userProcessedAfterCutoff = await TestUserRepository.CreateUserWithToken(nonMatchingTimestamp);
        var userWithoutTokenId = await TestUserRepository.CreateUser();
        
        Cleaner.AddCleanAction(async () =>
        {
            await TestCleaners.DeleteUsers([
                userNotProcessed.UserId,
                userProcessedBeforeCutoff.UserId,
                userProcessedAfterCutoff.UserId,
                userWithoutTokenId]);
        });
        
        //act
        var res = await StateTrackingGrpcClient.GetUsersForProcessingAsync(
            new GetUsersForProcessingRequest
            {
                BatchSize = 20,
                MinSecondsSinceLastProcess = notProcessedForMinutesThreshold * 60,
                ServiceType = ServiceType.Gmail
            }, cancellationToken: CancellationToken.None);
        
        //assert
        var idsForProcess = res.Users.Select(x => x.UserId).ToHashSet();
        idsForProcess.Contains(userNotProcessed.UserId.ToString()).Should().BeTrue();
        idsForProcess.Contains(userProcessedBeforeCutoff.UserId.ToString()).Should().BeTrue();
        idsForProcess.Contains(userProcessedAfterCutoff.UserId.ToString()).Should().BeFalse();
        idsForProcess.Contains(userWithoutTokenId.ToString()).Should().BeFalse();
        
        var notProcessedUserRes = res.Users.Single(x => x.UserId == userNotProcessed.UserId.ToString());
        notProcessedUserRes.Tokens.AccessToken.Should().Be(userNotProcessed.Token.AccessToken);
        notProcessedUserRes.Tokens.RefreshToken.Should().Be(userNotProcessed.Token.RefreshToken);
        notProcessedUserRes.Tokens.GmailEmail.Should().Be(userNotProcessed.Token.GmailEmail);
        notProcessedUserRes.Tokens.ExpiresAt.ToDateTime().Should().BeCloseTo(userNotProcessed.Token.ExpiresAt, TimeSpan.FromMilliseconds(100));
        
        var processedBeforeCutoffRes = res.Users.Single(x => x.UserId == userProcessedBeforeCutoff.UserId.ToString());
        processedBeforeCutoffRes.Tokens.AccessToken.Should().Be(userProcessedBeforeCutoff.Token.AccessToken);
        processedBeforeCutoffRes.Tokens.RefreshToken.Should().Be(userProcessedBeforeCutoff.Token.RefreshToken);
        processedBeforeCutoffRes.Tokens.GmailEmail.Should().Be(userProcessedBeforeCutoff.Token.GmailEmail);
        processedBeforeCutoffRes.Tokens.ExpiresAt.ToDateTime().Should().BeCloseTo(userProcessedBeforeCutoff.Token.ExpiresAt, TimeSpan.FromMilliseconds(100));
    }

    [TestMethod]
    public async Task GetUsersForProcessing_MoreUsersMatchThanLimit_ShouldOnlyReturnBatchSizeCountOfUsers()
    {
        //arrange
        const int matchingUsersCount = 5;
        const int batchSize = matchingUsersCount - 2;
        const int notProcessedForMinutesThreshold = 15;

        var users = new List<(Guid UserId, GmailToken Token)>();
        for (var i = 0; i < matchingUsersCount; ++i)
        {
            var matchingTimestamp = DateTime.UtcNow
                .AddMinutes(-notProcessedForMinutesThreshold - Random.Shared.Next(5, 100));
            var matchingUser = await TestUserRepository.CreateUserWithToken(matchingTimestamp);
            users.Add(matchingUser);
        }
        
        Cleaner.AddCleanAction(async () =>
        {
            await TestCleaners.DeleteUsers(users.Select(x => x.UserId).ToArray());
        });
        
        //act
        var res = await StateTrackingGrpcClient.GetUsersForProcessingAsync(
            new GetUsersForProcessingRequest
            {
                BatchSize = batchSize,
                MinSecondsSinceLastProcess = notProcessedForMinutesThreshold * 60,
                ServiceType = ServiceType.Gmail
            }, cancellationToken: CancellationToken.None);
        
        //assert
        res.Users.Should().HaveCount(batchSize);
    }

    [TestMethod]
    public async Task MarkUsersAsProcessed_ShouldMarkUsersAsProcessed()
    {
        //arrange
        const int notProcessedForMinutesThreshold = 15;
        var matchingTimestamp = DateTime.UtcNow
            .AddMinutes(-notProcessedForMinutesThreshold - Random.Shared.Next(5, 100));
        var user = await TestUserRepository.CreateUserWithToken(matchingTimestamp);
        
        Cleaner.AddCleanAction(async () => await TestCleaners.DeleteUser(user.UserId));
        
        //act
        var usersForProcessBeforeMarking = await StateTrackingGrpcClient.GetUsersForProcessingAsync(
            new GetUsersForProcessingRequest
            {
                BatchSize = 10000, //get all users
                MinSecondsSinceLastProcess = notProcessedForMinutesThreshold * 60,
                ServiceType = ServiceType.Gmail
            }, cancellationToken: CancellationToken.None);

        await StateTrackingGrpcClient.MarkUsersAsProcessedAsync(
            new MarkUsersAsProcessedRequest
            {
                ServiceType = ServiceType.Gmail,
                Users =
                {
                    new ProcessedUser
                    {
                        UserId = user.UserId.ToString(),
                        ProcessedAt = Timestamp.FromDateTime(DateTime.UtcNow)
                    }
                }
            }, cancellationToken: CancellationToken.None);
        
        var usersForProcessAfterMarking = await StateTrackingGrpcClient.GetUsersForProcessingAsync(
            new GetUsersForProcessingRequest
            {
                BatchSize = 10000, //get all users
                MinSecondsSinceLastProcess = notProcessedForMinutesThreshold * 60,
                ServiceType = ServiceType.Gmail
            }, cancellationToken: CancellationToken.None);
        
        //assert
        usersForProcessBeforeMarking.Users.Should().ContainSingle(x => x.UserId == user.UserId.ToString());
        usersForProcessAfterMarking.Users.Should().NotContain(x => x.UserId == user.UserId.ToString());
    }
}