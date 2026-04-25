using Personage.Auth.Domain.Models.Common;

namespace Personage.Auth.Domain.Interfaces;

public interface IIntegrationsService
{
    Task RevokeAccess(ServiceTypeModel serviceType, CancellationToken ct);
}