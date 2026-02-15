from app.domain.models.traits.AttachmentTrait import AttachmentTrait
from app.domain.models.traits.RecipientTrait import RecipientTrait
from app.domain.models.traits.SenderTrait import SenderTrait
from app.domain.models.traits.SubjectTrait import SubjectTrait

TraitUnion = AttachmentTrait | RecipientTrait | SenderTrait | SubjectTrait
