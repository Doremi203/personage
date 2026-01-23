from abc import ABC, abstractmethod

from app.domain.models.traits.base.TraitKindModel import TraitKindModel


class TraitModel(ABC):
    @property
    @abstractmethod
    def kind(self) -> TraitKindModel:
        pass
