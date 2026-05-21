namespace Personage.Auth.Api.Contracts.TelegramChats;

public class UpdateUserChatRequest
{
    public long ChatId { get; init; }
    public bool IsActive { get; init; }
}
