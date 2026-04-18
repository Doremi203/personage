package notification

import (
	"context"

	"github.com/google/uuid"
)

// Repo defines the persistence operations for notifications and notification settings.
type Repo interface {
	// Create inserts a new notification into the database.
	// Prefer CreateIfAbsent when the caller has an idempotency key.
	Create(ctx context.Context, n Notification) error

	// CreateIfAbsent inserts a new notification. When n.IdempotencyKey is
	// non-empty and a row with the same key already exists, returns
	// inserted=false and a nil error. An empty IdempotencyKey behaves like
	// Create (always inserts, inserted=true on success).
	CreateIfAbsent(ctx context.Context, n Notification) (inserted bool, err error)

	// ListByUserID returns a paginated list of notifications for the given user,
	// ordered by sent_at descending. offset is zero-based.
	ListByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]Notification, error)

	// ToggleSetting flips the enabled flag for the given user and notification type.
	// If no row exists yet, it inserts one with enabled=false (toggled from the default true).
	// Returns the new state after toggling.
	ToggleSetting(ctx context.Context, userID uuid.UUID, notificationType string) (Setting, error)

	// GetSettings returns all notification settings rows for the given user.
	GetSettings(ctx context.Context, userID uuid.UUID) ([]Setting, error)
}
