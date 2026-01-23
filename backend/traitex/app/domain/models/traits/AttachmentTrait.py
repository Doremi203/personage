from dataclasses import dataclass
from app.domain.models.traits.base.TraitKindModel import TraitKindModel
from app.domain.models.traits.base.TraitModel import TraitModel


@dataclass(frozen=True)
class Attachment:
    name: str


@dataclass(frozen=True)
class AttachmentTrait(TraitModel):
    @property
    def kind(self) -> TraitKindModel:
        return TraitKindModel.Attachment

    attachments: list[Attachment]
