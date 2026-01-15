using Personage.Auth.Domain.Models.Common;

namespace Personage.Auth.Domain.Models.StateTracking.Requests;

public class MarkUsersAsProcessedRequestModel
{
    public ServiceTypeModel ServiceType { get; init; }
    public ProcessedUserModel[] Users { get; init; } = [];
}