using System.Text.Json.Serialization;

namespace Personage.Auth.Api.Contracts.Integrations;

[JsonConverter(typeof(JsonStringEnumConverter))]
public enum IntegrationType
{
    Gmail = 1,
    GoogleCalendar = 2,
    Telegram = 3
}