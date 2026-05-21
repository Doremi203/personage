namespace Personage.Auth.Api.Contracts.TelegramChats;

public class TelegramChatDto
{
    public long ChatId { get; init; }
    public string ChatName { get; init; } = null!;
    public bool IsActive { get; init; }
}
