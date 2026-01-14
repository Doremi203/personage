using Personage.Auth.Domain.Models.StateTracking.Requests;
using Personage.Auth.Domain.Models.StateTracking.Responses;

namespace Personage.Auth.Domain.Interfaces;

public interface IStateTrackingService
{
    Task<GetUsersForProcessingResponseModel> GetUsersForProcessing(GetUsersForProcessingRequestModel request, CancellationToken ct);
    Task MarkUsersAsProcessed(MarkUsersAsProcessedRequestModel request, CancellationToken ct);
}