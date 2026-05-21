package pausepostgres

import (
	"context"
	"time"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/Doremi203/personage/backend/libs/go/postgres"
	"github.com/Doremi203/personage/backend/tasker/internal/domain"
)

func NewRepo(client postgres.Client, clock func() time.Time) *repo {
	return &repo{
		client: client,
		clock:  clock,
	}
}

type repo struct {
	client postgres.Client
	clock  func() time.Time
}

func (r *repo) IsPaused(ctx context.Context, userID domain.UserID) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1
			FROM processing_pauses
			WHERE user_id = $1
			AND (paused_until IS NULL OR paused_until > $2)
		)
	`

	var paused bool
	err := r.client.QueryRow(ctx, query, userID, r.clock()).Scan(&paused)
	if err != nil {
		return false, errors.WrapFail(err, "query processing pause")
	}

	return paused, nil
}
