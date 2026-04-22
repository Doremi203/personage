package notification

import (
	"time"

	"github.com/google/uuid"
)

type NotificationStatus string

const (
	StatusSent    NotificationStatus = "sent"
	StatusPending NotificationStatus = "pending"
	StatusDropped NotificationStatus = "dropped"
)

type PushPayload struct {
	Body string `json:"body"`
	Icon string `json:"icon"`
	URL  string `json:"url"`
}

// Notification represents a notification stored in the database.
type Notification struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	Title       string
	Type        string
	Text        string
	Status      NotificationStatus
	SentAt      *time.Time
	RetryAfter  *time.Time
	ExpiresAt   *time.Time
	PushPayload *PushPayload
	// IdempotencyKey dedupes writes. Empty means "no dedup, always insert".
	IdempotencyKey string
}
