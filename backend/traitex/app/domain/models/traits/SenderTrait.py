from dataclasses import dataclass

from app.domain.models.traits.base.TraitKindModel import TraitKindModel
from app.domain.models.traits.base.TraitModel import TraitModel
from app.domain.models.traits.common.UserIdentifier import UserIdentifier


@dataclass(frozen=True)
class SenderTrait(TraitModel):
    @property
    def kind(self) -> TraitKindModel:
        return TraitKindModel.Sender

    identifier: UserIdentifier
