package taskpostgres

import (
	"context"
	"fmt"
	"time"

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

func (r *repo) GetTasksByUserID(ctx context.Context, userID domain.UserID) ([]domain.Task, error) {
	query := `
		SELECT 
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
		FROM tasks
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.client.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("query tasks by user_id: %w", err)
	}
	defer rows.Close()

	var tasks []domain.Task
	for rows.Next() {
		var task domain.Task
		var durationMinutes int

		err := rows.Scan(
			&task.ID,
			&task.UserID,
			&task.ClusterID,
			&task.Title,
			&task.Description,
			&durationMinutes,
			&task.Priority,
			&task.Deadline,
			&task.StartTime,
			&task.Status,
			&task.CreatedAt,
			&task.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}

		task.Duration = domain.TimeSlotSize * time.Duration(durationMinutes/int(domain.TimeSlotSize.Minutes()))
		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tasks: %w", err)
	}

	return tasks, nil
}

func (r *repo) GetTasksByStatus(ctx context.Context, userID domain.UserID, status domain.TaskStatus) ([]domain.Task, error) {
	query := `
		SELECT 
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
		FROM tasks
		WHERE user_id = $1 AND status = $2
		ORDER BY created_at DESC
	`

	rows, err := r.client.Query(ctx, query, userID, status)
	if err != nil {
		return nil, fmt.Errorf("query tasks by status: %w", err)
	}
	defer rows.Close()

	var tasks []domain.Task
	for rows.Next() {
		var task domain.Task
		var durationMinutes int

		err := rows.Scan(
			&task.ID,
			&task.UserID,
			&task.ClusterID,
			&task.Title,
			&task.Description,
			&durationMinutes,
			&task.Priority,
			&task.Deadline,
			&task.StartTime,
			&task.Status,
			&task.CreatedAt,
			&task.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}

		task.Duration = domain.TimeSlotSize * time.Duration(durationMinutes/int(domain.TimeSlotSize.Minutes()))
		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tasks: %w", err)
	}

	return tasks, nil
}

func (r *repo) UpdateTaskSchedule(ctx context.Context, taskID domain.TaskID, startTime time.Time, status domain.TaskStatus) error {
	query := `
		UPDATE tasks
		SET start_time = $1,
			status = $2,
			updated_at = NOW()
		WHERE task_id = $3
	`

	result, err := r.client.Exec(ctx, query, startTime, status, taskID)
	if err != nil {
		return fmt.Errorf("update task schedule: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("task not found: %s", taskID)
	}

	return nil
}

func (r *repo) UpdateTaskStatus(ctx context.Context, taskID domain.TaskID, status domain.TaskStatus) error {
	query := `
		UPDATE tasks
		SET status = $1,
			updated_at = NOW()
		WHERE task_id = $2
	`

	result, err := r.client.Exec(ctx, query, status, taskID)
	if err != nil {
		return fmt.Errorf("update task status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("task not found: %s", taskID)
	}

	return nil
}
