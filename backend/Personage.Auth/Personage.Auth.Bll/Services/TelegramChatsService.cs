using Microsoft.Extensions.Logging;
using Personage.Auth.DataAccess.Interfaces.Repositories;
using Personage.Auth.Domain.Exceptions.Base;
using Personage.Auth.Domain.Interfaces;
using Personage.Auth.Domain.Models.TelegramAuth;

namespace Personage.Auth.Bll.Services;

public class TelegramChatsService(
    ITelegramSessionRepository telegramSessionRepository,
    ITelegramChatRepository telegramChatRepository,
    ITelegramChatsGrpcClient grpcClient,
    ILogger<TelegramChatsService> logger
) : ITelegramChatsService
{
    public async Task<IReadOnlyList<TelegramChatModel>> GetUserChats(Guid userId, CancellationToken ct)
    {
        var sessionString = await telegramSessionRepository.GetSessionString(userId, ct);
        if (sessionString is null)
            throw new NotFoundException(
                ErrorCode.TelegramSessionNotFound,
                $"Telegram session not found for user {userId}");

        var fetched = await grpcClient.GetUserChats(sessionString, ct);

        var existing = await telegramChatRepository.GetUserChats(userId, ct);
        var existingIds = existing.Select(c => c.ChatId).ToHashSet();

        var newOnes = fetched
            .Where(c => !existingIds.Contains(c.Id))
            .Select(c => (c.Id, c.Name))
            .ToArray();

        if (newOnes.Length == 0)
            return existing;

        logger.LogInformation(
            "Inserting {Count} new Telegram chats for user {UserId}",
            newOnes.Length, userId);
        await telegramChatRepository.InsertNewChats(userId, newOnes, ct);
        return await telegramChatRepository.GetUserChats(userId, ct);
    }

    public async Task UpdateActiveStatus(Guid userId, long chatId, bool isActive, CancellationToken ct)
    {
        var updated = await telegramChatRepository.UpdateActiveStatus(userId, chatId, isActive, ct);
        if (!updated)
            throw new NotFoundException(
                ErrorCode.TelegramChatNotFound,
                $"Telegram chat {chatId} not found for user {userId}");
    }
}
