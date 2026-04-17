package notificationpostgres

import (
	"database/sql"
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
	SentAt         time.Time      `db:"sent_at"`
	IdempotencyKey sql.NullString `db:"idempotency_key"`
}

func entityToDomain(e notificationEntity) notification.Notification {
	return notification.Notification{
		ID:             e.ID,
		UserID:         e.RecipientID,
		Title:          e.Title,
		Type:           e.Type,
		Text:           e.Text,
		SentAt:         e.SentAt,
		IdempotencyKey: e.IdempotencyKey.String,
	}
}

func domainToEntity(n notification.Notification) notificationEntity {
	return notificationEntity{
		RecipientID:    n.UserID,
		Title:          n.Title,
		Type:           n.Type,
		Text:           n.Text,
		IdempotencyKey: sql.NullString{String: n.IdempotencyKey, Valid: n.IdempotencyKey != ""},
	}
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
