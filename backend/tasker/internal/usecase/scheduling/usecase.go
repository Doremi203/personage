package scheduling

import (
	"context"
	"time"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/Doremi203/personage/backend/libs/go/log"
	"github.com/Doremi203/personage/backend/tasker/internal/domain"
	"github.com/Doremi203/personage/backend/tasker/internal/services/scheduler"
)

func NewUseCase(
	logger log.Logger,
	taskRepo domain.TaskRepo,
	notifier domain.NotificationsService,
	windowDuration time.Duration,
) *UseCase {
	return &UseCase{
		taskRepo:       taskRepo,
		notifier:       notifier,
		windowDuration: windowDuration,
		logger:         logger,
	}
}

type UseCase struct {
	taskRepo       domain.TaskRepo
	notifier       domain.NotificationsService
	windowDuration time.Duration
	logger         log.Logger
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

	uc.logger.Infof(
		"scheduling %s for user %s",
		errors.Token("pending_tasks", pendingTasks),
		errors.Token("user_id", userID.String()),
	)

	now := time.Now().Truncate(domain.TimeSlotSize)
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

	if len(schedule.Planned) > 0 {
		err := uc.notifier.Send(ctx, domain.Notification{
			UserID: userID,
			Title:  "📅 Ваше расписание изменилось",
			Body:   "Задачи были перепланированы, посмотрите в приложении...",
			Type:   "schedule_change",
		})
		if err != nil {
			uc.logger.Error(errors.WrapFail(err, "notify schedule changes"))
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
