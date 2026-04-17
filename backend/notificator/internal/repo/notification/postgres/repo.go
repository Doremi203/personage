package notificationpostgres

import (
	"context"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/Doremi203/personage/backend/libs/go/postgres"
	"github.com/Doremi203/personage/backend/libs/go/slices"
	"github.com/Doremi203/personage/backend/notificator/internal/domain/notification"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// NewRepo creates a new postgres-backed notification repository.
func NewRepo(db postgres.Client) *repo {
	return &repo{db: db}
}

type repo struct {
	db postgres.Client
}

func (r *repo) Create(ctx context.Context, n notification.Notification) error {
	const query = `
		INSERT INTO notifications (recipient_id, title, type, text, idempotency_key)
		VALUES ($1, $2, $3, $4, NULLIF($5, ''))
	`

	e := domainToEntity(n)

	_, err := r.db.Exec(ctx, query, e.RecipientID, e.Title, e.Type, e.Text, n.IdempotencyKey)
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
		INSERT INTO notifications (recipient_id, title, type, text, idempotency_key)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (idempotency_key) WHERE idempotency_key IS NOT NULL DO NOTHING
		RETURNING id
	`

	e := domainToEntity(n)

	rows, err := r.db.Query(ctx, query, e.RecipientID, e.Title, e.Type, e.Text, n.IdempotencyKey)
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
		SELECT id, recipient_id, title, type, text, sent_at, idempotency_key
		FROM notifications
		WHERE recipient_id = $1
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

	stored := make(map[notification.SettingType]notification.Setting, len(entities))
	for _, e := range entities {
		s := settingEntityToDomain(e)
		stored[s.Type] = s
	}

	result := make([]notification.Setting, 0, len(notification.AvailableSettingTypes))
	for _, typ := range notification.AvailableSettingTypes {
		if s, ok := stored[typ]; ok {
			result = append(result, s)
		} else {
			result = append(result, notification.Setting{
				UserID:  userID,
				Type:    typ,
				Enabled: true,
			})
		}
	}

	return result, nil
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
