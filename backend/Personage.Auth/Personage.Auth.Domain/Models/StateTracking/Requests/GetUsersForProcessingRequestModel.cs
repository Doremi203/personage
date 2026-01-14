using Personage.Auth.Domain.Models.Common;

namespace Personage.Auth.Domain.Models.StateTracking.Requests;

public class GetUsersForProcessingRequestModel
{
    public int BatchSize { get; init; }
    public int MinSecondsSinceLastProcess { get; init; }
    public ServiceTypeModel ServiceType { get; init; }
}
