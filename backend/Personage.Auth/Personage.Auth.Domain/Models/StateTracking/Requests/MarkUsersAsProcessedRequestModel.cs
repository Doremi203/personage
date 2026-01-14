namespace Personage.Auth.Domain.Models.StateTracking.Requests;

public class MarkUsersAsProcessedRequestModel
{
    public ProcessedUserModel[] Users { get; init; } = [];
}