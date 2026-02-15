from pydantic import BaseModel
from datetime import datetime
from uuid import UUID
from app.domain.models.ConnectorTypeModel import ConnectorTypeModel
from app.domain.models.traits.TraitUnion import TraitUnion


class EnrichedEventModel(BaseModel):
    id: UUID
    user_id: UUID
    connector_type: ConnectorTypeModel
    occurred_at: datetime
    main_body: str
    traits: list[TraitUnion]
