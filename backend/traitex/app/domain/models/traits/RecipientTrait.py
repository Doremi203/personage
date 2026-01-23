from dataclasses import dataclass
from app.domain.models.traits.base.TraitKindModel import TraitKindModel
from app.domain.models.traits.base.TraitModel import TraitModel
from app.domain.models.traits.common.UserIdentifier import UserIdentifier


@dataclass(frozen=True)
class RecipientTrait(TraitModel):
    @property
    def kind(self) -> TraitKindModel:
        return TraitKindModel.Recipient

    recipients: list[UserIdentifier]
