from app.domain.models.traits.base.TraitKindModel import TraitKindModel
from app.domain.models.traits.base.TraitModel import TraitModel
from app.domain.models.traits.common.UserIdentifier import UserIdentifier


class RecipientTrait(TraitModel):
    kind: TraitKindModel = TraitKindModel.Recipient
    recipients: list[UserIdentifier]
