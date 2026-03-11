package tasklist

import (
	"context"
	"time"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/Doremi203/personage/backend/libs/go/tx"
	"github.com/Doremi203/personage/backend/tasker/internal/domain"
)

const (
	dateFormat = "02-01-2006" // DD-MM-YYYY
)

type taskRepo interface {
	GetTaskByID(ctx context.Context, taskID domain.TaskID, userID domain.UserID) (domain.Task, error)
	UpdateTask(ctx context.Context, taskID domain.TaskID, userID domain.UserID, update domain.TaskUpdate) (domain.Task, error)
	UpdateTaskStatus(ctx context.Context, taskID domain.TaskID, status domain.TaskStatus) error
	DeleteTask(ctx context.Context, taskID domain.TaskID) error
	ListTasks(ctx context.Context, filter domain.TaskFilter, pagination domain.Pagination) ([]domain.Task, int, error)
}

type eventRepo interface {
	DeleteEventsByClusterID(ctx context.Context, clusterID domain.ClusterID) error
}

type clusterRepo interface {
	DeleteCluster(ctx context.Context, clusterID domain.ClusterID) error
}

type UseCase struct {
	taskRepo    taskRepo
	eventRepo   eventRepo
	clusterRepo clusterRepo
	txProvider  tx.Provider
}

func NewUseCase(
	taskRepo taskRepo,
	eventRepo eventRepo,
	clusterRepo clusterRepo,
	txProvider tx.Provider,
) *UseCase {
	return &UseCase{
		taskRepo:    taskRepo,
		eventRepo:   eventRepo,
		clusterRepo: clusterRepo,
		txProvider:  txProvider,
	}
}

type ListTasksParams struct {
	UserID   string
	Status   *domain.TaskStatus   // nil means "all"
	Category *domain.TaskCategory // nil means "all"
	Text     string
	From     string // date in DD-MM-YYYY format (validated by proto regex)
	Till     string // date in DD-MM-YYYY format (validated by proto regex)
	Page     int32
	PageSize int32
}

type ListTasksResult struct {
	Tasks    []domain.Task
	Total    int
	Page     int
	PageSize int
}

func (uc *UseCase) GetTask(ctx context.Context, taskID string, userID string) (domain.Task, error) {
	task, err := uc.taskRepo.GetTaskByID(ctx, domain.TaskID(taskID), domain.UserID(userID))
	if err != nil {
		return domain.Task{}, errors.WrapFail(err, "get task by id")
	}

	return task, nil
}

func (uc *UseCase) UpdateTask(ctx context.Context, taskID string, userID string, update domain.TaskUpdate) (domain.Task, error) {
	task, err := uc.taskRepo.UpdateTask(ctx, domain.TaskID(taskID), domain.UserID(userID), update)
	if err != nil {
		return domain.Task{}, errors.WrapFail(err, "update task")
	}

	return task, nil
}

// PostponeTask sets the task status to unplanned, verifying ownership first.
func (uc *UseCase) PostponeTask(ctx context.Context, taskID string, userID string) (domain.Task, error) {
	task, err := uc.taskRepo.GetTaskByID(ctx, domain.TaskID(taskID), domain.UserID(userID))
	if err != nil {
		return domain.Task{}, errors.WrapFail(err, "postpone task")
	}

	if err := uc.taskRepo.UpdateTaskStatus(ctx, domain.TaskID(taskID), domain.TaskStatusUnplanned); err != nil {
		return domain.Task{}, errors.WrapFail(err, "postpone task")
	}
	task.Status = domain.TaskStatusUnplanned

	return task, nil
}

// CompleteTask sets the task status to completed, verifying ownership first.
func (uc *UseCase) CompleteTask(ctx context.Context, taskID string, userID string) (domain.Task, error) {
	task, err := uc.taskRepo.GetTaskByID(ctx, domain.TaskID(taskID), domain.UserID(userID))
	if err != nil {
		return domain.Task{}, errors.WrapFail(err, "complete task")
	}

	if err := uc.taskRepo.UpdateTaskStatus(ctx, domain.TaskID(taskID), domain.TaskStatusCompleted); err != nil {
		return domain.Task{}, errors.WrapFail(err, "complete task")
	}
	task.Status = domain.TaskStatusCompleted

	return task, nil
}

// DeleteTask deletes a task and its associated events and cluster within a transaction.
func (uc *UseCase) DeleteTask(ctx context.Context, taskID string, userID string) error {
	// Verify ownership and get cluster_id.
	task, err := uc.taskRepo.GetTaskByID(ctx, domain.TaskID(taskID), domain.UserID(userID))
	if err != nil {
		return errors.WrapFail(err, "delete task")
	}

	return uc.txProvider.RunWithTx(ctx, tx.IsolationReadCommitted, func(txCtx context.Context) error {
		// 1. Delete events referencing the cluster.
		if err := uc.eventRepo.DeleteEventsByClusterID(txCtx, task.ClusterID); err != nil {
			return errors.WrapFail(err, "delete events for task")
		}

		// 2. Delete the task (references the cluster).
		if err := uc.taskRepo.DeleteTask(txCtx, task.ID); err != nil {
			return errors.WrapFail(err, "delete task record")
		}

		// 3. Delete the cluster.
		if err := uc.clusterRepo.DeleteCluster(txCtx, task.ClusterID); err != nil {
			return errors.WrapFail(err, "delete cluster for task")
		}

		return nil
	})
}

func (uc *UseCase) GetTasks(ctx context.Context, params ListTasksParams) (ListTasksResult, error) {
	filter := domain.TaskFilter{
		UserID:   domain.UserID(params.UserID),
		Status:   params.Status,
		Category: params.Category,
		Text:     params.Text,
	}

	if params.From != "" {
		from, err := time.Parse(dateFormat, params.From)
		if err != nil {
			return ListTasksResult{}, errors.Wrap(err, "invalid from date only format")
		}
		filter.From = &from
	}

	if params.Till != "" {
		till, err := time.Parse(dateFormat, params.Till)
		if err != nil {
			return ListTasksResult{}, errors.Wrap(err, "invalid till date only format")
		}
		endOfDay := till.Add(24 * time.Hour)
		filter.Till = &endOfDay
	}

	pagination := domain.Pagination{
		Page:     int(params.Page),
		PageSize: int(params.PageSize),
	}

	tasks, total, err := uc.taskRepo.ListTasks(ctx, filter, pagination)
	if err != nil {
		return ListTasksResult{}, errors.WrapFail(err, "list tasks")
	}

	return ListTasksResult{
		Tasks:    tasks,
		Total:    total,
		Page:     pagination.Page,
		PageSize: pagination.PageSize,
	}, nil
}
