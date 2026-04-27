package notificationpostgres

import (
	"context"
	"time"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/Doremi203/personage/backend/libs/go/postgres"
	"github.com/Doremi203/personage/backend/libs/go/slices"
	"github.com/Doremi203/personage/backend/notificator/internal/domain/notification"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func NewRepo(db postgres.Client) *repo {
	return &repo{db: db}
}

type repo struct {
	db postgres.Client
}

func (r *repo) Create(ctx context.Context, n notification.Notification) error {
	const query = `
		INSERT INTO notifications (recipient_id, title, type, text, status, sent_at, retry_after, expires_at, push_payload, idempotency_key)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NULLIF($10, ''))
	`
	e := domainToEntity(n)
	_, err := r.db.Exec(ctx, query,
		e.RecipientID, e.Title, e.Type, e.Text,
		e.Status, e.SentAt, e.RetryAfter, e.ExpiresAt, e.PushPayload,
		n.IdempotencyKey,
	)
	if err != nil {
		return errors.WrapFail(err, "exec insert notification query")
	}
	return nil
}

func (r *repo) CreateIfAbsent(ctx context.Context, n notification.Notification) (bool, error) {
	if n.IdempotencyKey == "" {
		if err := r.Create(ctx, n); err != nil {
			return false, err
		}
		return true, nil
	}

	const query = `
		INSERT INTO notifications (recipient_id, title, type, text, status, sent_at, retry_after, expires_at, push_payload, idempotency_key)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (idempotency_key) WHERE idempotency_key IS NOT NULL DO NOTHING
		RETURNING id
	`
	e := domainToEntity(n)
	rows, err := r.db.Query(ctx, query,
		e.RecipientID, e.Title, e.Type, e.Text,
		e.Status, e.SentAt, e.RetryAfter, e.ExpiresAt, e.PushPayload,
		n.IdempotencyKey,
	)
	if err != nil {
		return false, errors.WrapFail(err, "exec insert-if-absent notification query")
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return false, errors.WrapFail(err, "iterate insert-if-absent rows")
		}
		return false, nil
	}
	return true, nil
}

func (r *repo) ListByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]notification.Notification, error) {
	const query = `
		SELECT id, recipient_id, title, type, text, status, sent_at, retry_after, expires_at, read_at, push_payload, idempotency_key
		FROM notifications
		WHERE recipient_id = $1 AND status = 'sent'
		ORDER BY sent_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, errors.WrapFail(err, "exec select notifications query")
	}
	defer rows.Close()

	entities, err := pgx.CollectRows(rows, pgx.RowToStructByName[notificationEntity])
	if err != nil {
		return nil, errors.WrapFail(err, "collect notification rows")
	}

	return slices.Map(entities, entityToDomain), nil
}

func (r *repo) IsSettingEnabled(ctx context.Context, userID uuid.UUID, typ notification.SettingType) (bool, error) {
	const query = `
		SELECT enabled
		FROM notification_settings
		WHERE recipient_id = $1 AND type = $2
	`
	rows, err := r.db.Query(ctx, query, userID, string(typ))
	if err != nil {
		return false, errors.WrapFail(err, "exec select notification setting query")
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return false, errors.WrapFail(err, "iterate notification setting rows")
		}
		return true, nil
	}
	var enabled bool
	if err := rows.Scan(&enabled); err != nil {
		return false, errors.WrapFail(err, "scan notification setting enabled")
	}
	return enabled, nil
}

func (r *repo) GetSettings(ctx context.Context, userID uuid.UUID) ([]notification.Setting, error) {
	const query = `
		SELECT recipient_id, type, enabled
		FROM notification_settings
		WHERE recipient_id = $1
	`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, errors.WrapFail(err, "exec select notification settings query")
	}
	defer rows.Close()

	entities, err := pgx.CollectRows(rows, pgx.RowToStructByName[settingEntity])
	if err != nil {
		return nil, errors.WrapFail(err, "collect notification settings rows")
	}

	return slices.Map(entities, settingEntityToDomain), nil
}

func (r *repo) CreateAndReturnID(ctx context.Context, n notification.Notification) (uuid.UUID, error) {
	const query = `
		INSERT INTO notifications (recipient_id, title, type, text)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`

	e := domainToEntity(n)

	rows, err := r.db.Query(ctx, query, e.RecipientID, e.Title, e.Type, e.Text)
	if err != nil {
		return uuid.Nil, errors.WrapFail(err, "exec insert notification query")
	}
	defer rows.Close()

	id, err := pgx.CollectOneRow(rows, func(row pgx.CollectableRow) (uuid.UUID, error) {
		var id uuid.UUID
		return id, row.Scan(&id)
	})
	if err != nil {
		return uuid.Nil, errors.WrapFail(err, "collect created notification id")
	}

	return id, nil
}

func (r *repo) MarkAsRead(ctx context.Context, id, userID uuid.UUID, readAt time.Time) error {
	const query = `
		UPDATE notifications
		SET read_at = COALESCE(read_at, $1)
		WHERE id = $2 AND recipient_id = $3 AND status = 'sent'
	`
	tag, err := r.db.Exec(ctx, query, readAt, id, userID)
	if err != nil {
		return errors.WrapFail(err, "exec mark notification read query")
	}
	if tag.RowsAffected() == 0 {
		return notification.ErrNotificationNotFound
	}
	return nil
}

func (r *repo) MarkAllAsRead(ctx context.Context, userID uuid.UUID, readAt time.Time) error {
	const query = `
		UPDATE notifications
		SET read_at = $1
		WHERE recipient_id = $2 AND status = 'sent' AND read_at IS NULL
	`
	_, err := r.db.Exec(ctx, query, readAt, userID)
	if err != nil {
		return errors.WrapFail(err, "exec mark all notifications read query")
	}
	return nil
}

func (r *repo) ToggleSetting(ctx context.Context, userID uuid.UUID, notificationType string) (notification.Setting, error) {
	const query = `
		INSERT INTO notification_settings (recipient_id, type, enabled)
		VALUES ($1, $2, false)
		ON CONFLICT (recipient_id, type) DO UPDATE
		SET enabled = NOT notification_settings.enabled
		RETURNING recipient_id, type, enabled
	`
	rows, err := r.db.Query(ctx, query, userID, notificationType)
	if err != nil {
		return notification.Setting{}, errors.WrapFail(err, "exec toggle notification setting query")
	}
	defer rows.Close()

	entity, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[settingEntity])
	if err != nil {
		return notification.Setting{}, errors.WrapFail(err, "collect toggle setting row")
	}

	return settingEntityToDomain(entity), nil
}

func (r *repo) ListPending(ctx context.Context) ([]notification.Notification, error) {
	const query = `
		SELECT id, recipient_id, title, type, text, status, sent_at, retry_after, expires_at, read_at, push_payload, idempotency_key
		FROM notifications
		WHERE status = 'pending'
	`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, errors.WrapFail(err, "exec select pending notifications query")
	}
	defer rows.Close()

	entities, err := pgx.CollectRows(rows, pgx.RowToStructByName[notificationEntity])
	if err != nil {
		return nil, errors.WrapFail(err, "collect pending notification rows")
	}

	return slices.Map(entities, entityToDomain), nil
}

func (r *repo) Drop(ctx context.Context, id uuid.UUID) error {
	const query = `UPDATE notifications SET status = 'dropped' WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return errors.WrapFail(err, "exec drop notification query")
	}
	return nil
}

func (r *repo) MarkSent(ctx context.Context, id uuid.UUID, sentAt time.Time) error {
	const query = `UPDATE notifications SET status = 'sent', sent_at = $1 WHERE id = $2`
	_, err := r.db.Exec(ctx, query, sentAt, id)
	if err != nil {
		return errors.WrapFail(err, "exec mark notification sent query")
	}
	return nil
}

func (r *repo) UpdateRetryAfter(ctx context.Context, id uuid.UUID, retryAfter time.Time) error {
	const query = `UPDATE notifications SET retry_after = $1 WHERE id = $2`
	_, err := r.db.Exec(ctx, query, retryAfter, id)
	if err != nil {
		return errors.WrapFail(err, "exec update notification retry_after query")
	}
	return nil
}

func (r *repo) CountSentSince(ctx context.Context, userID uuid.UUID, typ notification.SettingType, since time.Time) (int, error) {
	const query = `
		SELECT COUNT(*)
		FROM notifications
		WHERE recipient_id = $1
		  AND type = $2
		  AND status = 'sent'
		  AND sent_at > $3
	`
	rows, err := r.db.Query(ctx, query, userID, string(typ), since)
	if err != nil {
		return 0, errors.WrapFail(err, "exec count sent notifications query")
	}
	defer rows.Close()

	var count int
	if rows.Next() {
		if err = rows.Scan(&count); err != nil {
			return 0, errors.WrapFail(err, "scan count")
		}
	}
	return count, nil
}
