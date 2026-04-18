package notificationpostgres

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/Doremi203/personage/backend/notificator/internal/domain/notification"
	"github.com/google/uuid"
)

type notificationEntity struct {
	ID             uuid.UUID      `db:"id"`
	RecipientID    uuid.UUID      `db:"recipient_id"`
	Title          string         `db:"title"`
	Type           string         `db:"type"`
	Text           string         `db:"text"`
	Status         string         `db:"status"`
	SentAt         *time.Time     `db:"sent_at"`
	RetryAfter     *time.Time     `db:"retry_after"`
	ExpiresAt      *time.Time     `db:"expires_at"`
	PushPayload    *string        `db:"push_payload"`
	IdempotencyKey sql.NullString `db:"idempotency_key"`
}

func entityToDomain(e notificationEntity) notification.Notification {
	n := notification.Notification{
		ID:             e.ID,
		UserID:         e.RecipientID,
		Title:          e.Title,
		Type:           e.Type,
		Text:           e.Text,
		Status:         notification.NotificationStatus(e.Status),
		SentAt:         e.SentAt,
		RetryAfter:     e.RetryAfter,
		ExpiresAt:      e.ExpiresAt,
		IdempotencyKey: e.IdempotencyKey.String,
	}
	if e.PushPayload != nil {
		var p notification.PushPayload
		if err := json.Unmarshal([]byte(*e.PushPayload), &p); err == nil {
			n.PushPayload = &p
		}
	}
	return n
}

func domainToEntity(n notification.Notification) notificationEntity {
	e := notificationEntity{
		RecipientID:    n.UserID,
		Title:          n.Title,
		Type:           n.Type,
		Text:           n.Text,
		Status:         string(n.Status),
		SentAt:         n.SentAt,
		RetryAfter:     n.RetryAfter,
		ExpiresAt:      n.ExpiresAt,
		IdempotencyKey: sql.NullString{String: n.IdempotencyKey, Valid: n.IdempotencyKey != ""},
	}
	if n.PushPayload != nil {
		b, err := json.Marshal(n.PushPayload)
		if err == nil {
			s := string(b)
			e.PushPayload = &s
		}
	}
	return e
}

type settingEntity struct {
	RecipientID uuid.UUID `db:"recipient_id"`
	Type        string    `db:"type"`
	Enabled     bool      `db:"enabled"`
}

func settingEntityToDomain(e settingEntity) notification.Setting {
	return notification.Setting{
		UserID:  e.RecipientID,
		Type:    notification.SettingType(e.Type),
		Enabled: e.Enabled,
	}
}
