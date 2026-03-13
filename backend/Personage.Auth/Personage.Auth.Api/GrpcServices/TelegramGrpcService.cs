using Grpc.Core;
using Personage.Auth.Api.Grpc.Telegram;
using Personage.Auth.Domain.Interfaces;
using Personage.Auth.Domain.Models.TelegramAuth.Requests;

namespace Personage.Auth.Api.GrpcServices;

public class TelegramGrpcService(ITelegramAuthService telegramAuthService) : TelegramService.TelegramServiceBase
{
    public override async Task<StoreSessionResponse> StoreSession(StoreSessionRequest request, ServerCallContext context)
    {
        await telegramAuthService.StoreSession(new StoreSessionRequestModel
        {
            UserId = Guid.Parse(request.UserId),
            SessionString = request.SessionString
        }, context.CancellationToken);

        return new StoreSessionResponse();
    }

    public override async Task<GetSessionResponse> GetSession(GetSessionRequest request, ServerCallContext context)
    {
        var res = await telegramAuthService.GetSession(
            Guid.Parse(request.UserId),
            context.CancellationToken
        );

        return new GetSessionResponse
        {
            SessionString = res.SessionString
        };
    }
}