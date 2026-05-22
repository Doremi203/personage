package notification

import (
	"time"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/google/uuid"
)

var (
	ErrEmptyUserID = errors.Error("notification user id is empty")
	ErrEmptyTitle  = errors.Error("notification title is empty")
	ErrEmptyType   = errors.Error("notification type is empty")
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
	ReadAt      *time.Time // nil when the recipient has not opened the notification yet
	PushPayload *PushPayload
	// IdempotencyKey dedupes writes. Empty means "no dedup, always insert".
	IdempotencyKey string
}

func validateCommon(userID uuid.UUID, title, typ string) error {
	if userID == uuid.Nil {
		return ErrEmptyUserID
	}
	if title == "" {
		return ErrEmptyTitle
	}
	if typ == "" {
		return ErrEmptyType
	}
	return nil
}

func NewPending(
	userID uuid.UUID,
	title, typ, text string,
	retryAfter, expiresAt time.Time,
	payload *PushPayload,
	idempotencyKey string,
) (Notification, error) {
	if err := validateCommon(userID, title, typ); err != nil {
		return Notification{}, err
	}
	return Notification{
		UserID:         userID,
		Title:          title,
		Type:           typ,
		Text:           text,
		Status:         StatusPending,
		RetryAfter:     &retryAfter,
		ExpiresAt:      &expiresAt,
		PushPayload:    payload,
		IdempotencyKey: idempotencyKey,
	}, nil
}

func NewSent(
	userID uuid.UUID,
	title, typ, text string,
	sentAt time.Time,
	idempotencyKey string,
) (Notification, error) {
	if err := validateCommon(userID, title, typ); err != nil {
		return Notification{}, err
	}
	return Notification{
		UserID:         userID,
		Title:          title,
		Type:           typ,
		Text:           text,
		Status:         StatusSent,
		SentAt:         &sentAt,
		IdempotencyKey: idempotencyKey,
	}, nil
}
