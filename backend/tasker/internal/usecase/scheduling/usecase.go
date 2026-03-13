package scheduling

import (
	"context"
	"time"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/Doremi203/personage/backend/libs/go/log"
	"github.com/Doremi203/personage/backend/tasker/internal/domain"
	"github.com/Doremi203/personage/backend/tasker/internal/services/scheduler"
)

type taskRepo interface {
	GetUsersWithUnplannedTasks(ctx context.Context) ([]domain.UserID, error)
	GetTasksByStatus(ctx context.Context, userID domain.UserID, status domain.TaskStatus) ([]domain.Task, error)
	UpdateTaskSchedule(ctx context.Context, taskID domain.TaskID, startTime time.Time, endTime time.Time, status domain.TaskStatus) error
}

type UseCase struct {
	taskRepo       taskRepo
	windowDuration time.Duration
	logger         log.Logger
}

func NewUseCase(
	taskRepo taskRepo,
	windowDuration time.Duration,
	logger log.Logger,
) *UseCase {
	return &UseCase{
		taskRepo:       taskRepo,
		windowDuration: windowDuration,
		logger:         logger,
	}
}

func (uc *UseCase) SchedulePendingTasks(ctx context.Context) error {
	userIDs, err := uc.taskRepo.GetUsersWithUnplannedTasks(ctx)
	if err != nil {
		return errors.WrapFail(err, "get users with pending tasks")
	}

	if len(userIDs) == 0 {
		return nil
	}

	for _, userID := range userIDs {
		if err := uc.scheduleForUser(ctx, userID); err != nil {
			uc.logger.Error(errors.WrapFailf(
				err,
				"schedule tasks for user %s",
				errors.Token("user_id", userID.String()),
			))
			continue
		}
	}

	return nil
}

func (uc *UseCase) scheduleForUser(ctx context.Context, userID domain.UserID) error {
	pendingTasks, err := uc.taskRepo.GetTasksByStatus(ctx, userID, domain.TaskStatusUnplanned)
	if err != nil {
		return errors.WrapFail(err, "get pending tasks")
	}

	if len(pendingTasks) == 0 {
		return nil
	}

	now := time.Now()
	schedule := scheduler.CalculateSchedule(pendingTasks, now, uc.windowDuration)

	for _, planned := range schedule.Planned {
		if err := uc.taskRepo.UpdateTaskSchedule(
			ctx,
			planned.ID,
			planned.Start,
			planned.End,
			domain.TaskStatusPlanned,
		); err != nil {
			return errors.WrapFailf(
				err,
				"update task schedule %s",
				errors.Token("task_id", planned.ID.String()),
			)
		}
	}

	uc.logger.Infof(
		"scheduled tasks for user %s %s %s",
		errors.Token("user_id", userID.String()),
		errors.Token("scheduled_count", len(schedule.Planned)),
		errors.Token("unscheduled_count", len(schedule.Unscheduled)),
	)

	return nil
}
