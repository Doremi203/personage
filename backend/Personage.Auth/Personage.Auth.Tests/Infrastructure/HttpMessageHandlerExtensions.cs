using System;
using System.Net.Http;
using System.Threading;
using System.Threading.Tasks;
using Moq;
using Moq.Protected;

namespace Personage.Auth.Tests.Infrastructure;

public static class HttpMessageHandlerMockExtensions
{
    public static void SetupSendAsync(
        this Mock<HttpMessageHandler> handler,
        string requestUrl,
        HttpMethod method,
        string responseContent,
        string contentType = "application/json",
        Func<HttpRequestMessage, bool>? requestValidator = null,
        System.Net.HttpStatusCode statusCode = System.Net.HttpStatusCode.OK)
    {
        handler.Protected()
            .Setup<Task<HttpResponseMessage>>(
                "SendAsync",
                ItExpr.Is<HttpRequestMessage>(req =>
                    req.RequestUri!.ToString() == requestUrl &&
                    req.Method == method &&
                    (requestValidator == null || requestValidator(req))),
                ItExpr.IsAny<CancellationToken>())
            .ReturnsAsync(new HttpResponseMessage
            {
                StatusCode = statusCode,
                Content = new StringContent(responseContent, System.Text.Encoding.UTF8, contentType)
            });
    }

    public static void VerifySendAsync(
        this Mock<HttpMessageHandler> handler,
        string requestUrl,
        HttpMethod method,
        Times times,
        Func<string, bool>? contentValidator = null)
    {
        handler.Protected().Verify(
            "SendAsync",
            times,
            ItExpr.Is<HttpRequestMessage>(req =>
                req.RequestUri!.ToString() == requestUrl &&
                req.Method == method &&
                (contentValidator == null ||
                 contentValidator(req.Content!.ReadAsStringAsync().GetAwaiter().GetResult()))),
            ItExpr.IsAny<CancellationToken>());
    }
}