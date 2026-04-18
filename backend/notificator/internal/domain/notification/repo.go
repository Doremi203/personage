package notification

import (
	"context"
	"time"

	"github.com/google/uuid"
)

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

	ListPending(ctx context.Context) ([]Notification, error)
	Drop(ctx context.Context, id uuid.UUID) error
	MarkSent(ctx context.Context, id uuid.UUID, sentAt time.Time) error
	UpdateRetryAfter(ctx context.Context, id uuid.UUID, retryAfter time.Time) error
	CountSentSince(ctx context.Context, userID uuid.UUID, typ SettingType, since time.Time) (int, error)
}
