from app.domain.exceptions.base.BusinessErrorCode import BusinessErrorCode
from app.domain.exceptions.base.BusinessException import BusinessException
from app.domain.models.traits.base.TraitKindModel import TraitKindModel


class DuplicateTraitEncountered(BusinessException):
    def __init__(
            self,
            trait_kind: TraitKindModel
    ):
        message = f'Found more than one trait of kind {trait_kind}'
        super().__init__(BusinessErrorCode.DuplicateTraitEncountered, message)
