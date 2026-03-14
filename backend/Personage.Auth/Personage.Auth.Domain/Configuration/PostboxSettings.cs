namespace Personage.Auth.Domain.Configuration;

public class PostboxSettings
{
    public string FromEmail { get; init; } = null!;
    public string FromName { get; init; } = "Personage Auth";

    /// <summary>
    /// Yandex Cloud service account access key ID for Postbox SMTP authentication.
    /// In production, resolved from Lockbox via <c>secret:{id}:{version}:{key}</c>.
    /// </summary>
    public string AccessKeyId { get; set; } = null!;

    /// <summary>
    /// Yandex Cloud service account secret key for Postbox SMTP authentication.
    /// In production, resolved from Lockbox via <c>secret:{id}:{version}:{key}</c>.
    /// </summary>
    public string SecretKey { get; set; } = null!;
}
