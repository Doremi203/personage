package taskpostgres

import (
	"context"
	"fmt"

	"gitlab.com/amoguscorp/personage/backend/libs/go/postgres"
	"gitlab.com/amoguscorp/personage/backend/tasker/internal/domain"
)

func NewRepo(client postgres.Client) *repo {
	return &repo{
		client: client,
	}
}

type repo struct {
	client postgres.Client
}

func (r *repo) CreateTask(ctx context.Context, task domain.Task) error {
	query := `
		INSERT INTO tasks (
			task_id,
			user_id,
			cluster_id,
			title,
			description,
			duration_minutes,
			priority,
			deadline,
			start_time,
			status,
			created_at,
			updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		)
	`

	_, err := r.client.Exec(ctx, query,
		task.ID,
		task.UserID,
		task.ClusterID,
		task.Title,
		task.Description,
		int(task.Duration.Minutes()),
		task.Priority,
		task.Deadline,
		task.StartTime,
		task.Status,
		task.CreatedAt,
		task.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("create task: %w", err)
	}

	return nil
}
