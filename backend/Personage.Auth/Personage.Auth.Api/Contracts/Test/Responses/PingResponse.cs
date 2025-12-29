namespace Personage.Auth.Contracts.Test.Responses;

public class PingResponse
{
    public string Message { get; init; } = null!;
    public DateTimeOffset Moment { get; init; }
};