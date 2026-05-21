package moderationpostgres

import (
	"context"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/Doremi203/personage/backend/libs/go/postgres"
	"github.com/Doremi203/personage/backend/tasker/internal/domain"
	"github.com/jackc/pgx/v5"
)

func NewRepo(client postgres.Client) *repo {
	return &repo{
		client: client,
	}
}

type repo struct {
	client postgres.Client
}

func (r *repo) RequiresModeration(ctx context.Context, userID domain.UserID) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1
			FROM manual_moderation_users
			WHERE user_id = $1
		)
	`

	var required bool
	err := r.client.QueryRow(ctx, query, userID).Scan(&required)
	if err != nil {
		return false, errors.WrapFail(err, "query manual moderation user")
	}

	return required, nil
}

func (r *repo) AddUser(ctx context.Context, userID domain.UserID) error {
	query := `
		INSERT INTO manual_moderation_users (user_id)
		VALUES ($1)
		ON CONFLICT (user_id) DO NOTHING
	`

	_, err := r.client.Exec(ctx, query, userID)
	if err != nil {
		return errors.WrapFail(err, "insert manual moderation user")
	}

	return nil
}

func (r *repo) RemoveUser(ctx context.Context, userID domain.UserID) error {
	query := `DELETE FROM manual_moderation_users WHERE user_id = $1`

	_, err := r.client.Exec(ctx, query, userID)
	if err != nil {
		return errors.WrapFail(err, "delete manual moderation user")
	}

	return nil
}

func (r *repo) ListUsers(ctx context.Context) ([]domain.UserID, error) {
	query := `SELECT user_id FROM manual_moderation_users ORDER BY created_at DESC`

	rows, err := r.client.Query(ctx, query)
	if err != nil {
		return nil, errors.WrapFail(err, "query manual moderation users")
	}
	defer rows.Close()

	userIDs, err := pgx.CollectRows(rows, pgx.RowTo[domain.UserID])
	if err != nil {
		return nil, errors.WrapFail(err, "collect manual moderation user rows")
	}

	return userIDs, nil
}
