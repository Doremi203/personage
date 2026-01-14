using Personage.Auth.Domain.Interfaces;
using Personage.Auth.Domain.Models.StateTracking.Requests;
using Personage.Auth.Domain.Models.StateTracking.Responses;

namespace Personage.Auth.Bll.Services;

public class StateTrackingService : IStateTrackingService
{
    public Task<GetUsersForProcessingResponseModel> GetUsersForProcessing(GetUsersForProcessingRequestModel request, CancellationToken ct)
    {
        throw new NotImplementedException();
    }

    public Task MarkUsersAsProcessed(MarkUsersAsProcessedRequestModel request, CancellationToken ct)
    {
        throw new NotImplementedException();
    }
}