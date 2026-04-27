using Personage.Auth.DataAccess.Interfaces.Repositories;
using Personage.Auth.Domain.Exceptions;
using Personage.Auth.Domain.Interfaces;
using Personage.Auth.Domain.Models.Common;

namespace Personage.Auth.Bll.Services;

public class IntegrationsService(
    IClaimValues claimValues,
    IGmailTokenRepository gmailTokenRepository,
    IGoogleCalendarTokenRepository googleCalendarTokenRepository,
    ITelegramSessionRepository telegramSessionRepository
) : IIntegrationsService
{
    public async Task RevokeAccess(ServiceTypeModel serviceType, CancellationToken ct)
    {
        var userId = claimValues.GetUserId();

        switch (serviceType)
        {
            case ServiceTypeModel.Gmail:
                await gmailTokenRepository.RemoveToken(userId, ct);
                break;
            case ServiceTypeModel.Telegram:
                await telegramSessionRepository.RemoveSession(userId, ct);
                break;
            case ServiceTypeModel.GoogleCalendar:
                await googleCalendarTokenRepository.RemoveToken(userId, ct);
                break;
            case ServiceTypeModel.Unknown:
            default:
                throw new ServiceTypeNotSupportedException($"Cannot revoke access for service type {serviceType}");
        }
    }
}