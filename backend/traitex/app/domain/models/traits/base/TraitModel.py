from pydantic import BaseModel
from app.domain.models.traits.base.TraitKindModel import TraitKindModel


class TraitModel(BaseModel):
    kind: TraitKindModel
