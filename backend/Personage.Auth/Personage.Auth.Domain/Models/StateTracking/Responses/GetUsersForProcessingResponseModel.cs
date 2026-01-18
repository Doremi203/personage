namespace Personage.Auth.Domain.Models.StateTracking.Responses;

public class GetUsersForProcessingResponseModel
{
    public UserForProcessingModel[] Users { get; init; } = [];
}
