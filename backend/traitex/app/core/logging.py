import json
import logging
import sys
from datetime import datetime, timezone
from typing import Any

_LEVEL_MAP: dict[int, str] = {
    logging.DEBUG: "DEBUG",
    logging.INFO: "INFO",
    logging.WARNING: "WARN",
    logging.ERROR: "ERROR",
    logging.CRITICAL: "FATAL",
}


class JSONFormatter(logging.Formatter):
    # Keys that belong to the standard LogRecord and should NOT be forwarded
    # as extra structured fields.
    _BUILTIN_ATTRS: frozenset[str] = frozenset({
        "args", "created", "exc_info", "exc_text", "filename", "funcName",
        "levelname", "levelno", "lineno", "module", "msecs", "message", "msg",
        "name", "pathname", "process", "processName", "relativeCreated",
        "stack_info", "thread", "threadName", "taskName",
    })

    def format(self, record: logging.LogRecord) -> str:
        # Build the base payload matching Go slog output shape.
        payload: dict[str, Any] = {
            "time": datetime.fromtimestamp(record.created, tz=timezone.utc)
                        .strftime("%Y-%m-%dT%H:%M:%S.%f")[:-3] + "Z",
            "level": _LEVEL_MAP.get(record.levelno, "ERROR"),
            "msg": record.getMessage(),
            "logger": record.name,
        }

        for key, value in record.__dict__.items():
            if key not in self._BUILTIN_ATTRS and key not in payload:
                payload[key] = value

        # Append exception info when present.
        if record.exc_info and record.exc_info[1] is not None:
            payload["error"] = self.formatException(record.exc_info)

        if record.stack_info:
            payload["stack"] = record.stack_info

        return json.dumps(payload, ensure_ascii=False, default=str)


def setup_logging(logging_config: dict[str, Any]) -> None:
    level_name: str = logging_config.get("Level", "INFO")
    level: int = getattr(logging, level_name.upper(), logging.INFO)

    fmt_value: str = logging_config.get("Format", "console")

    root = logging.getLogger()
    root.setLevel(level)

    # Remove any previously attached handlers (e.g. from basicConfig).
    for handler in root.handlers[:]:
        root.removeHandler(handler)

    handler = logging.StreamHandler(sys.stdout)
    handler.setLevel(level)

    if fmt_value.lower() == "json":
        handler.setFormatter(JSONFormatter())
    else:
        if "%" in fmt_value:
            pattern = fmt_value
        else:
            pattern = "%(asctime)s - %(name)s - %(levelname)s - %(message)s"
        handler.setFormatter(logging.Formatter(pattern))

    root.addHandler(handler)
