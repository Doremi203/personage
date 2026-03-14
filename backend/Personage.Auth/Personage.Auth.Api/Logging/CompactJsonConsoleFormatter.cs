using System.Globalization;
using System.Text.Json;
using Microsoft.Extensions.Logging;
using Microsoft.Extensions.Logging.Abstractions;
using Microsoft.Extensions.Logging.Console;

namespace Personage.Auth.Api.Logging;

/// <summary>
/// A compact JSON console log formatter that outputs log entries in a format compatible
/// with the shared FluentBit pipeline. Field names match Go slog's JSON handler output
/// (<c>time</c>, <c>level</c>, <c>msg</c>) so that the FluentBit parser and
/// yc-logging output plugin can extract message text and severity without extra remapping.
/// </summary>
public sealed class CompactJsonConsoleFormatter : ConsoleFormatter
{
    public const string FormatterName = "compact-json";

    public CompactJsonConsoleFormatter()
        : base(FormatterName)
    {
    }

    public override void Write<TState>(
        in LogEntry<TState> logEntry,
        IExternalScopeProvider? scopeProvider,
        TextWriter textWriter)
    {
        var message = logEntry.Formatter?.Invoke(logEntry.State, logEntry.Exception);
        if (message is null)
            return;

        using var stream = new MemoryStream();
        using (var writer = new Utf8JsonWriter(stream, new JsonWriterOptions { Indented = false }))
        {
            writer.WriteStartObject();

            // "time" — ISO 8601 UTC timestamp, matching Go slog format.
            writer.WriteString("time",
                DateTimeOffset.UtcNow.ToString("yyyy-MM-ddTHH:mm:ss.fffZ", CultureInfo.InvariantCulture));

            // "level" — uppercase severity matching Yandex Cloud Logging expectations
            // (DEBUG, INFO, WARN, ERROR, FATAL).
            writer.WriteString("level", MapLogLevel(logEntry.LogLevel));

            // "msg" — the formatted log message.
            writer.WriteString("msg", message);

            // "category" — the logger category (typically the fully-qualified class name).
            writer.WriteString("category", logEntry.Category);

            // Include exception details when present.
            if (logEntry.Exception is not null)
            {
                writer.WriteString("error", logEntry.Exception.ToString());
            }

            // Include event ID when it is non-default.
            if (logEntry.EventId.Id != 0)
            {
                writer.WriteNumber("event_id", logEntry.EventId.Id);
            }

            writer.WriteEndObject();
        }

        // Write the JSON line followed by a newline — one JSON object per line.
        textWriter.Write(System.Text.Encoding.UTF8.GetString(stream.ToArray()));
        textWriter.WriteLine();
    }

    /// <summary>
    /// Maps .NET <see cref="LogLevel"/> values to uppercase severity strings
    /// compatible with Yandex Cloud Logging and the FluentBit yc-logging plugin.
    /// </summary>
    private static string MapLogLevel(LogLevel logLevel) => logLevel switch
    {
        LogLevel.Trace => "TRACE",
        LogLevel.Debug => "DEBUG",
        LogLevel.Information => "INFO",
        LogLevel.Warning => "WARN",
        LogLevel.Error => "ERROR",
        LogLevel.Critical => "FATAL",
        LogLevel.None => "DEFAULT",
        _ => "WARN"
    };
}
