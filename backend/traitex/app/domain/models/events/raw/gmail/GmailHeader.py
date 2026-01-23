from dataclasses import dataclass


@dataclass(frozen=True)
class GmailMessageHeader:
    name: str
    value: str
