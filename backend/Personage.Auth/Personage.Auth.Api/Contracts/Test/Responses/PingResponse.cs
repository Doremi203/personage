namespace Personage.Auth.Api.Contracts.Test.Responses;

public class PingResponse
{
    public string Message { get; init; } = null!;
    public DateTimeOffset Moment { get; init; }
};