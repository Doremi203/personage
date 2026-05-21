package moderationpostgres

import (
	"context"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/Doremi203/personage/backend/libs/go/postgres"
	"github.com/Doremi203/personage/backend/tasker/internal/domain"
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
