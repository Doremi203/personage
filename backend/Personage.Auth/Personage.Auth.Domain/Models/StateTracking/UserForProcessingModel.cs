using Personage.Auth.Domain.Models.Common;

namespace Personage.Auth.Domain.Models.StateTracking;

public class UserForProcessingModel
{
    public Guid UserId { get; init; }
    public string UserEmail { get; init; } = null!;
    public string GmailEmail { get; init; } = null!;
    public GmailTokenModel Tokens { get; init; } = null!;
    public DateTime? LastProcessedAt { get; init; }
}
