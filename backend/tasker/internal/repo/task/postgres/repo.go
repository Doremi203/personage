package taskpostgres

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/Doremi203/personage/backend/libs/go/postgres"
	"github.com/Doremi203/personage/backend/libs/go/slices"
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
			end_time,
			status,
			category,
			created_at,
			updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
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
		task.EndTime,
		task.Status,
		task.Category,
		task.CreatedAt,
		task.UpdatedAt,
	)

	if err != nil {
		return errors.WrapFail(err, "create task")
	}

	return nil
}

func (r *repo) GetTaskByID(ctx context.Context, taskID domain.TaskID, userID domain.UserID) (domain.Task, error) {
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
			end_time,
			status,
			category,
			created_at,
			updated_at
		FROM tasks
		WHERE task_id = $1 AND user_id = $2
	`

	rows, err := r.client.Query(ctx, query, taskID, userID)
	if err != nil {
		return domain.Task{}, errors.WrapFail(err, "query task by id")
	}
	defer rows.Close()

	entity, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[taskEntity])
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.Task{}, domain.ErrTaskNotFound
		}
		return domain.Task{}, errors.WrapFail(err, "collect task row")
	}

	return entity.ToDomain(), nil
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
			end_time,
			status,
			category,
			created_at,
			updated_at
		FROM tasks
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.client.Query(ctx, query, userID)
	if err != nil {
		return nil, errors.WrapFail(err, "query tasks by user_id")
	}
	defer rows.Close()

	entities, err := pgx.CollectRows(rows, pgx.RowToStructByName[taskEntity])
	if err != nil {
		return nil, errors.WrapFail(err, "collect task rows")
	}

	return slices.Map(entities, taskEntity.ToDomain), nil
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
			end_time,
			status,
			category,
			created_at,
			updated_at
		FROM tasks
		WHERE user_id = $1 AND status = $2
		ORDER BY created_at DESC
	`

	rows, err := r.client.Query(ctx, query, userID, status)
	if err != nil {
		return nil, errors.WrapFail(err, "query tasks by status")
	}
	defer rows.Close()

	entities, err := pgx.CollectRows(rows, pgx.RowToStructByName[taskEntity])
	if err != nil {
		return nil, errors.WrapFail(err, "collect task rows")
	}

	return slices.Map(entities, taskEntity.ToDomain), nil
}

func (r *repo) GetUsersWithUnplannedTasks(ctx context.Context) ([]domain.UserID, error) {
	query := `
		SELECT DISTINCT user_id
		FROM tasks
		WHERE status = $1
	`

	rows, err := r.client.Query(ctx, query, domain.TaskStatusUnplanned)
	if err != nil {
		return nil, errors.WrapFail(err, "query users with unplanned tasks")
	}
	defer rows.Close()

	userIDs, err := pgx.CollectRows(rows, pgx.RowTo[domain.UserID])
	if err != nil {
		return nil, errors.WrapFail(err, "collect user_id rows")
	}

	return userIDs, nil
}

func (r *repo) UpdateTaskSchedule(ctx context.Context, taskID domain.TaskID, startTime time.Time, endTime time.Time, status domain.TaskStatus) error {
	query := `
		UPDATE tasks
		SET start_time = $1,
			end_time = $2,
			status = $3,
			updated_at = NOW()
		WHERE task_id = $4
	`

	result, err := r.client.Exec(ctx, query, startTime, endTime, status, taskID)
	if err != nil {
		return errors.WrapFail(err, "update task schedule")
	}

	if result.RowsAffected() == 0 {
		return errors.Errorf("task not found: %s", taskID)
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
		return errors.WrapFail(err, "update task status")
	}

	if result.RowsAffected() == 0 {
		return errors.Errorf("task not found: %s", taskID)
	}

	return nil
}

func (r *repo) DeleteTask(ctx context.Context, taskID domain.TaskID) error {
	query := `DELETE FROM tasks WHERE task_id = $1`

	result, err := r.client.Exec(ctx, query, taskID)
	if err != nil {
		return errors.WrapFail(err, "delete task")
	}

	if result.RowsAffected() == 0 {
		return domain.ErrTaskNotFound
	}

	return nil
}

func (r *repo) UpdateTask(ctx context.Context, taskID domain.TaskID, userID domain.UserID, update domain.TaskUpdate) (domain.Task, error) {
	var setClauses []string
	var args []any
	argIdx := 1

	if update.Title != nil {
		setClauses = append(setClauses, fmt.Sprintf("title = $%d", argIdx))
		args = append(args, *update.Title)
		argIdx++
	}

	if update.Description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *update.Description)
		argIdx++
	}

	if update.StartTime != nil {
		setClauses = append(setClauses, fmt.Sprintf("start_time = $%d", argIdx))
		args = append(args, *update.StartTime)
		argIdx++
	}

	if update.EndTime != nil {
		setClauses = append(setClauses, fmt.Sprintf("end_time = $%d", argIdx))
		args = append(args, *update.EndTime)
		argIdx++
	}

	if update.Category != nil {
		setClauses = append(setClauses, fmt.Sprintf("category = $%d", argIdx))
		args = append(args, string(*update.Category))
		argIdx++
	}

	if len(setClauses) == 0 {
		// Nothing to update — just return the current task.
		return r.GetTaskByID(ctx, taskID, userID)
	}

	setClauses = append(setClauses, "updated_at = NOW()")

	query := fmt.Sprintf(`
		UPDATE tasks
		SET %s
		WHERE task_id = $%d AND user_id = $%d
		RETURNING
			task_id,
			user_id,
			cluster_id,
			title,
			description,
			duration_minutes,
			priority,
			deadline,
			start_time,
			end_time,
			status,
			category,
			created_at,
			updated_at
	`, strings.Join(setClauses, ", "), argIdx, argIdx+1)

	args = append(args, taskID, userID)

	rows, err := r.client.Query(ctx, query, args...)
	if err != nil {
		return domain.Task{}, errors.WrapFail(err, "update task")
	}
	defer rows.Close()

	entity, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[taskEntity])
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.Task{}, domain.ErrTaskNotFound
		}
		return domain.Task{}, errors.WrapFail(err, "collect updated task row")
	}

	return entity.ToDomain(), nil
}

func (r *repo) ListTasks(ctx context.Context, filter domain.TaskFilter, pagination domain.Pagination) ([]domain.Task, int, error) {
	var conditions []string
	var args []any
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("user_id = $%d", argIdx))
	args = append(args, filter.UserID)
	argIdx++

	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *filter.Status)
		argIdx++
	}

	if filter.Category != nil {
		conditions = append(conditions, fmt.Sprintf("category = $%d", argIdx))
		args = append(args, *filter.Category)
		argIdx++
	}

	if filter.Text != "" {
		conditions = append(conditions, fmt.Sprintf("search_vector @@ plainto_tsquery('english', $%d)", argIdx))
		args = append(args, filter.Text)
		argIdx++
	}

	if filter.From != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argIdx))
		args = append(args, *filter.From)
		argIdx++
	}

	if filter.Till != nil {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argIdx))
		args = append(args, *filter.Till)
		argIdx++
	}

	whereClause := strings.Join(conditions, " AND ")

	countQuery := "SELECT COUNT(*) FROM tasks WHERE " + whereClause
	var total int
	err := r.client.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, errors.WrapFail(err, "count tasks")
	}

	offset := (pagination.Page - 1) * pagination.PageSize
	dataQuery := `
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
			end_time,
			status,
			category,
			created_at,
			updated_at
		FROM tasks
		WHERE ` + whereClause + `
		ORDER BY created_at DESC
		LIMIT $` + strconv.Itoa(argIdx) + ` OFFSET $` + strconv.Itoa(argIdx+1)

	args = append(args, pagination.PageSize, offset)

	rows, err := r.client.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, errors.WrapFail(err, "query tasks")
	}
	defer rows.Close()

	entities, err := pgx.CollectRows(rows, pgx.RowToStructByName[taskEntity])
	if err != nil {
		return nil, 0, errors.WrapFail(err, "collect task rows")
	}

	return slices.Map(entities, taskEntity.ToDomain), total, nil
}

type taskEntity struct {
	TaskID          string     `db:"task_id"`
	UserID          string     `db:"user_id"`
	ClusterID       string     `db:"cluster_id"`
	Title           string     `db:"title"`
	Description     string     `db:"description"`
	DurationMinutes int        `db:"duration_minutes"`
	Priority        int        `db:"priority"`
	Deadline        *time.Time `db:"deadline"`
	StartTime       *time.Time `db:"start_time"`
	EndTime         *time.Time `db:"end_time"`
	Status          string     `db:"status"`
	Category        string     `db:"category"`
	CreatedAt       time.Time  `db:"created_at"`
	UpdatedAt       time.Time  `db:"updated_at"`
}

func (e taskEntity) ToDomain() domain.Task {
	return domain.Task{
		ID:          domain.TaskID(e.TaskID),
		UserID:      domain.UserID(e.UserID),
		ClusterID:   domain.ClusterID(e.ClusterID),
		Title:       e.Title,
		Description: e.Description,
		Duration:    time.Duration(e.DurationMinutes) * time.Minute,
		Priority:    e.Priority,
		Deadline:    e.Deadline,
		StartTime:   e.StartTime,
		EndTime:     e.EndTime,
		Status:      domain.TaskStatus(e.Status),
		Category:    domain.TaskCategory(e.Category),
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
	}
}
