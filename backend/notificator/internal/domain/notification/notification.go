package notification

import (
	"time"

	"github.com/google/uuid"
)

// Notification represents a sent notification stored in the database.
type Notification struct {
	ID     uuid.UUID
	UserID uuid.UUID
	Title  string
	Type   string
	Text   string
	SentAt time.Time
	// IdempotencyKey dedupes writes. Empty means "no dedup, always insert".
	IdempotencyKey string
}
