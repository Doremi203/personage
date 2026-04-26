package notification

import (
	"context"
	"time"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/google/uuid"
)

// ErrNotificationNotFound is returned when a notification id does not exist
// for the requesting user (cross-user reads must not succeed).
var ErrNotificationNotFound = errors.Error("notification not found")

//go:generate mockgen -source=repo.go -destination=mock/repo_mock.go -typed

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
	ToggleSetting(ctx context.Context, userID uuid.UUID, notificationType string) (Setting, error)
	GetSettings(ctx context.Context, userID uuid.UUID) ([]Setting, error)

	// MarkAsRead sets read_at on a notification owned by userID. Idempotent:
	// a second call leaves the original read_at unchanged. Returns
	// ErrNotificationNotFound when the row does not exist for the user.
	MarkAsRead(ctx context.Context, id, userID uuid.UUID, readAt time.Time) error

	// MarkAllAsRead sets read_at on every currently-unread notification for
	// userID. Already-read notifications are left untouched.
	MarkAllAsRead(ctx context.Context, userID uuid.UUID, readAt time.Time) error

	ListPending(ctx context.Context) ([]Notification, error)
	Drop(ctx context.Context, id uuid.UUID) error
	MarkSent(ctx context.Context, id uuid.UUID, sentAt time.Time) error
	UpdateRetryAfter(ctx context.Context, id uuid.UUID, retryAfter time.Time) error
	CountSentSince(ctx context.Context, userID uuid.UUID, typ SettingType, since time.Time) (int, error)
}
