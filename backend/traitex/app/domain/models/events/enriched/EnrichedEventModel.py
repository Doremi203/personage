from dataclasses import dataclass
from datetime import datetime
from uuid import UUID
from app.domain.models import ConnectorTypeModel
from app.domain.models.traits.base.TraitModel import TraitModel


@dataclass(frozen=True)
class EnrichedEventModel:
    id: UUID
    user_id: UUID
    connector_type: ConnectorTypeModel
    occurred_at: datetime
    main_body: str
    traits: list[TraitModel]
