from enum import Enum


class TraitKindModel(str, Enum):
    Unknown = "unknown"
    Subject = "subject"
    Recipient = "recipient"
    Attachment = "attachment"
    Sender = "sender"
