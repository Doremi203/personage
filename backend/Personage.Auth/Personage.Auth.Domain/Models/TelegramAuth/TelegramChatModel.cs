namespace Personage.Auth.Domain.Models.TelegramAuth;

public class TelegramChatModel
{
    public long ChatId { get; init; }
    public string ChatName { get; init; } = null!;
    public bool IsActive { get; init; }
}
