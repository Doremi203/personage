namespace Personage.Auth.Domain.Models.StateTracking;

public class ProcessedUserModel
{
    public Guid UserId { get; set; }
    public DateTime ProcessedAt { get; set; }
}