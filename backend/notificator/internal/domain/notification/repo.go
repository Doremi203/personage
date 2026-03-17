package notification

import (
	"context"

	"github.com/google/uuid"
)

// Repo defines the persistence operations for notifications and notification settings.
type Repo interface {
	// Create inserts a new notification into the database.
	Create(ctx context.Context, n Notification) error

	// ListByUserID returns a paginated list of notifications for the given user,
	// ordered by sent_at descending. offset is zero-based.
	ListByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]Notification, error)

	// ToggleSetting flips the enabled flag for the given user and notification type.
	// If no row exists yet, it inserts one with enabled=false (toggled from the default true).
	// Returns the new state after toggling.
	ToggleSetting(ctx context.Context, userID uuid.UUID, notificationType string) (Setting, error)
}
