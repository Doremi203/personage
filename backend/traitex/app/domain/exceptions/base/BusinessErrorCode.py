from enum import Enum


class BusinessErrorCode(Enum):
    Unknown = 0
    DuplicateUserProcessingInfo = 1
    DuplicateTraitEncountered = 2
    SnapshotNotFound = 3

    def __str__(self) -> str    :
        return self.name
