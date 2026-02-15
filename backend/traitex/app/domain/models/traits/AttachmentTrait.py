from pydantic import BaseModel
from app.domain.models.traits.base.TraitKindModel import TraitKindModel
from app.domain.models.traits.base.TraitModel import TraitModel


class Attachment(BaseModel):
    name: str


class AttachmentTrait(TraitModel):
    kind: TraitKindModel = TraitKindModel.Attachment
    attachments: list[Attachment]
