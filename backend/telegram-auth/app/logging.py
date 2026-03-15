"""
Logging configuration for the Telegram Auth service.

Produces JSON output matching the Go slog / traitex JSONFormatter shape
in production, and human-readable console output in development.

JSON payload shape (production):
    {"time": "2026-03-15T11:12:07.123Z", "level": "INFO", "msg": "...", "logger": "...", ...}
"""

import logging
import sys

import structlog

_LEVEL_MAP: dict[str, str] = {
    "debug": "DEBUG",
    "info": "INFO",
    "warning": "WARN",
    "error": "ERROR",
    "critical": "FATAL",
}


def _rename_level(_, __, event_dict: dict) -> dict:
    """Map structlog level names to the Go slog convention (WARN, FATAL)."""
    level = event_dict.get("level", "")
    event_dict["level"] = _LEVEL_MAP.get(level, level.upper())
    return event_dict


def _rename_event_to_msg(_, __, event_dict: dict) -> dict:
    """Rename structlog's 'event' key to 'msg' to match Go slog output."""
    event_dict["msg"] = event_dict.pop("event", "")
    return event_dict


def _add_logger_name(_, __, event_dict: dict) -> dict:
    """Add 'logger' key from the stdlib logger name, matching traitex format."""
    record = event_dict.get("_record")
    if record:
        event_dict["logger"] = record.name
    return event_dict


def setup_logging(log_level: str = "INFO", log_format: str = "console") -> None:
    """Configure structlog + stdlib logging.

    Args:
        log_level: One of DEBUG, INFO, WARNING, ERROR.
        log_format: "json" for production (matches traitex JSONFormatter),
                    "console" for human-readable dev output.
    """
    level = getattr(logging, log_level.upper(), logging.INFO)
    is_json = log_format.lower() == "json"

    # Shared processors applied to every log entry.
    shared_processors: list[structlog.types.Processor] = [
        structlog.contextvars.merge_contextvars,
        structlog.stdlib.add_log_level,
        structlog.stdlib.add_logger_name,
        structlog.processors.TimeStamper(fmt="iso", utc=True),
        structlog.processors.StackInfoRenderer(),
        structlog.processors.UnicodeDecoder(),
    ]

    if is_json:
        # Processors that shape the dict before JSON serialisation.
        structlog_processors = [
            *shared_processors,
            # Merge stdlib LogRecord fields so we can extract logger name.
            structlog.stdlib.ProcessorFormatter.wrap_for_formatter,
        ]

        formatter_processors = [
            structlog.stdlib.ProcessorFormatter.remove_processors_meta,
            _rename_level,
            _add_logger_name,
            _rename_event_to_msg,
            structlog.processors.JSONRenderer(),
        ]
    else:
        structlog_processors = [
            *shared_processors,
            structlog.stdlib.ProcessorFormatter.wrap_for_formatter,
        ]

        formatter_processors = [
            structlog.stdlib.ProcessorFormatter.remove_processors_meta,
            structlog.dev.ConsoleRenderer(),
        ]

    # Configure structlog to use stdlib as the backend.
    structlog.configure(
        processors=structlog_processors,
        logger_factory=structlog.stdlib.LoggerFactory(),
        wrapper_class=structlog.stdlib.BoundLogger,
        cache_logger_on_first_use=True,
    )

    # Configure the stdlib root logger with our formatter.
    formatter = structlog.stdlib.ProcessorFormatter(
        processors=formatter_processors,
    )

    handler = logging.StreamHandler(sys.stdout)
    handler.setFormatter(formatter)

    root = logging.getLogger()
    root.handlers.clear()
    root.addHandler(handler)
    root.setLevel(level)

    # Quieten noisy third-party loggers.
    for noisy in ("uvicorn", "uvicorn.access", "uvicorn.error", "httpx", "httpcore"):
        logging.getLogger(noisy).setLevel(max(level, logging.WARNING))
