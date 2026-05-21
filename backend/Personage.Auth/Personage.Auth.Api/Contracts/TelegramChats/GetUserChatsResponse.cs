namespace Personage.Auth.Api.Contracts.TelegramChats;

public class GetUserChatsResponse
{
    public IReadOnlyList<TelegramChatDto> Chats { get; init; } = [];
}
