using Microsoft.Extensions.Options;
using Personage.Auth.Api.Grpc.TelegramChats;
using Personage.Auth.Domain.Configuration;
using Personage.Auth.Domain.Interfaces;
using GrpcClient = Personage.Auth.Api.Grpc.TelegramChats.TelegramChatsService.TelegramChatsServiceClient;

namespace Personage.Auth.Api.GrpcClients;

public class TelegramChatsGrpcClient(
    GrpcClient client,
    IOptions<TelegramAuthGrpcSettings> settings
) : ITelegramChatsGrpcClient
{
    public async Task<IReadOnlyList<(long Id, string Name)>> GetUserChats(
        string sessionString,
        CancellationToken ct)
    {
        var deadline = DateTime.UtcNow.AddSeconds(settings.Value.TimeoutSeconds);
        var response = await client.GetUserChatsAsync(
            new GetUserChatsRequest { SessionString = sessionString },
            deadline: deadline,
            cancellationToken: ct);

        return response.Chats
            .Select(c => (c.Id, c.Name))
            .ToList();
    }
}
