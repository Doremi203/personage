from enum import Enum


class ConnectorTypeModel(str, Enum):
    Unknown = "unknown"
    Gmail = "gmail"
    Telegram = "telegram"
    GoogleCalendar = "calendar"
