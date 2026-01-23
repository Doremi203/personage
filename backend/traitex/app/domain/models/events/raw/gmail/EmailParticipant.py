from dataclasses import dataclass


@dataclass(frozen=True)
class EmailParticipant:
    name: str
    email: str
