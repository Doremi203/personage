using System.Security.Cryptography;
using System.Text;
using MailKit.Net.Smtp;
using MailKit.Security;
using Microsoft.Extensions.Logging;
using Microsoft.Extensions.Options;
using MimeKit;
using Personage.Auth.Domain.Configuration;
using Personage.Auth.Domain.Interfaces;
using Personage.Auth.Domain.Models.Postbox.Requests;
using Yandex.Cloud;
using Yandex.Cloud.Lockbox.V1;

namespace Personage.Auth.Bll.Services;

public class PostboxService(
    IOptions<PostboxSettings> settings,
    ILogger<PostboxService> logger,
    Sdk yandexSdk
) : IPostboxService
{
    private string? _cachedAccessKeyId;
    private string? _cachedSecretKey;
    private DateTime _keysExpiry = DateTime.MinValue;

    public async Task SendEmailAsync(
        SendEmailRequestModel request,
        CancellationToken ct
    )
    {
        const string postboxHost = "postbox.cloud.yandex.net";
        const int postboxPort = 587;

        try
        {
            var (accessKeyId, secretKey) = await GetKeysFromLockboxAsync(ct);
            var smtpPassword = GenerateSmtpPassword(secretKey);

            var message = new MimeMessage();
            message.From.Add(new MailboxAddress(settings.Value.FromName, settings.Value.FromEmail));
            message.To.Add(MailboxAddress.Parse(request.To));
            message.Subject = request.Subject;
            message.Body = new TextPart("html") { Text = request.HtmlBody };

            using var client = new SmtpClient();
            await client.ConnectAsync(
                postboxHost,
                postboxPort,
                SecureSocketOptions.StartTls,
                ct
            );
            await client.AuthenticateAsync(
                accessKeyId,
                smtpPassword,
                ct
            );
            await client.SendAsync(message, ct);
            await client.DisconnectAsync(true, ct);

            logger.LogInformation("Email sent to {To}", request.To);
        }
        catch (Exception ex)
        {
            logger.LogError(ex, "Failed to send email to {To}", request.To);
            throw;
        }
    }

    public async Task SendPasswordResetEmail(
        string email,
        string resetUrl,
        CancellationToken ct
    )
    {
        const string subject = "Reset your password";
        //TODO: add proper password reset email content
        var htmlBody = $"""
                                    <h2>Password Reset Request</h2>
                                    <p>Click the link below to reset your password. This link will expire in 1 hour.</p>
                                    <p><a href='{resetUrl}'>Reset Password</a></p>
                                    <p>If you didn't request this, please ignore this email.</p>
                        """;

        await SendEmailAsync(new SendEmailRequestModel
        {
            To = email,
            Subject = subject,
            HtmlBody = htmlBody
        }, ct);
    }

    private async Task<(string AccessKeyId, string SecretKey)> GetKeysFromLockboxAsync(CancellationToken ct)
    {
        if (_keysExpiry > DateTime.UtcNow &&
            _cachedAccessKeyId != null &&
            _cachedSecretKey != null)
        {
            return (_cachedAccessKeyId, _cachedSecretKey);
        }

        try
        {
            var payload = await yandexSdk.Services.Lockbox.PayloadService.GetAsync(
                new GetPayloadRequest { SecretId = settings.Value.SecretId },
                cancellationToken: ct
            );

            _cachedAccessKeyId = payload.Entries.First(e => e.Key == "access_key_id").TextValue;
            _cachedSecretKey = payload.Entries.First(e => e.Key == "secret_key").TextValue;
            _keysExpiry = DateTime.UtcNow.AddHours(1);

            return (_cachedAccessKeyId, _cachedSecretKey);
        }
        catch (Exception ex)
        {
            logger.LogError(ex, "Failed to retrieve keys from Lockbox");
            throw;
        }
    }

    private static string GenerateSmtpPassword(string secretKey)
    {
        const string postboxFixedDate = "20230926";
        const string service = "postbox";
        const string message = "SendRawEmail";
        const string region = "ru-central1";
        const string terminal = "aws4_request";
        const byte version = 0x04;

        var signature = HmacSha256(Encoding.UTF8.GetBytes("AWS4" + secretKey), postboxFixedDate);
        signature = HmacSha256(signature, region);
        signature = HmacSha256(signature, service);
        signature = HmacSha256(signature, terminal);
        signature = HmacSha256(signature, message);

        var signatureWithVersion = new byte[signature.Length + 1];
        signatureWithVersion[0] = version;
        Array.Copy(signature, 0, signatureWithVersion, 1, signature.Length);

        return Convert.ToBase64String(signatureWithVersion);
    }

    private static byte[] HmacSha256(byte[] key, string data)
    {
        using var hmac = new HMACSHA256(key);
        return hmac.ComputeHash(Encoding.UTF8.GetBytes(data));
    }
}