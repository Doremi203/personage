from app.domain.models.traits.base.TraitKindModel import TraitKindModel
from app.domain.models.traits.base.TraitModel import TraitModel


class SubjectTrait(TraitModel):
    kind: TraitKindModel = TraitKindModel.Subject
    name: str
