from dataclasses import dataclass
from app.domain.models.traits.base.TraitKindModel import TraitKindModel
from app.domain.models.traits.base.TraitModel import TraitModel


@dataclass(frozen=True)
class SubjectTrait(TraitModel):
    @property
    def kind(self) -> TraitKindModel:
        return TraitKindModel.Subject

    name: str
