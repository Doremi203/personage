namespace Personage.Auth.Domain.Models.StateTracking;

public class UserForProcessingModel
{
    public Guid UserId { get; init; }
    public DateTime? LastProcessedAt { get; init; }
    public ProcessingCredentialsBase Credentials { get; init; } = null!;
}
