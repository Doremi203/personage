using System.Security.Cryptography;
using System.Text;
using MailKit.Net.Smtp;
using MailKit.Security;
using Microsoft.Extensions.Logging;
using Microsoft.Extensions.Options;
using MimeKit;
using Personage.Auth.Bll.Resources;
using Personage.Auth.Domain.Configuration;
using Personage.Auth.Domain.Interfaces;
using Personage.Auth.Domain.Models.Postbox.Requests;

namespace Personage.Auth.Bll.Services;

public class PostboxService(
    IOptions<PostboxSettings> settings,
    ILogger<PostboxService> logger
) : IPostboxService
{
    public async Task SendEmailAsync(
        SendEmailRequestModel request,
        CancellationToken ct
    )
    {
        const string postboxHost = "postbox.cloud.yandex.net";
        const int postboxPort = 587;

        try
        {
            var accessKeyId = settings.Value.AccessKeyId;
            var secretKey = settings.Value.SecretKey;
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
        var htmlBody = string.Format(EmailTemplates.PasswordResetHtml, resetUrl);

        await SendEmailAsync(new SendEmailRequestModel
        {
            To = email,
            Subject = EmailTemplates.PasswordResetSubject,
            HtmlBody = htmlBody
        }, ct);
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
