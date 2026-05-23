package scheduling

import (
	"context"
	"slices"
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
	clock func() time.Time,
	location *time.Location,
) *UseCase {
	return &UseCase{
		taskRepo:       taskRepo,
		notifier:       notifier,
		windowDuration: windowDuration,
		logger:         logger,
		clock:          clock,
		location:       location,
	}
}

type UseCase struct {
	taskRepo       domain.TaskRepo
	notifier       domain.NotificationsService
	windowDuration time.Duration
	logger         log.Logger
	clock          func() time.Time
	location       *time.Location
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

	now := uc.clock().Truncate(domain.TimeSlotSize)
	windowEnd := now.Add(uc.windowDuration)

	existingPlanned, err := uc.taskRepo.GetPlannedTasksInRange(ctx, userID, now, windowEnd)
	if err != nil {
		return errors.WrapFail(err, "get existing planned tasks")
	}

	uc.logger.Infof(
		"scheduling %s with %s existing planned for user %s",
		errors.Token("pending_tasks", pendingTasks),
		errors.Token("existing_planned_count", len(existingPlanned)),
		errors.Token("user_id", userID.String()),
	)

	unplannedIDs := make(map[domain.TaskID]struct{}, len(pendingTasks))
	approvedUnplannedIDs := make(map[domain.TaskID]struct{}, len(pendingTasks))
	for _, t := range pendingTasks {
		unplannedIDs[t.ID] = struct{}{}
		if t.IsApproved {
			approvedUnplannedIDs[t.ID] = struct{}{}
		}
	}

	inputTasks := slices.Concat(pendingTasks, existingPlanned)
	schedule := scheduler.CalculateSchedule(inputTasks, now, uc.windowDuration, uc.location)

	persistedCount := 0
	notifiableCount := 0
	for _, planned := range schedule.Planned {
		if _, isNew := unplannedIDs[planned.ID]; !isNew {
			continue
		}
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
		persistedCount++
		if _, ok := approvedUnplannedIDs[planned.ID]; ok {
			notifiableCount++
		}
	}

	if notifiableCount > 0 {
		err := uc.notifier.Send(ctx, domain.Notification{
			UserID: userID,
			Title:  "📅 Ваше расписание изменилось",
			Body:   "Задачи были перепланированы, посмотрите в приложении...",
			Type:   "schedule_change",
		})
		if err != nil {
			uc.logger.Error(errors.WrapFailf(
				err,
				"notify schedule changes for user %s",
				errors.Token("user_id", userID.String()),
			))
		}
	}

	uc.logger.Infof(
		"scheduled tasks for user %s %s %s",
		errors.Token("user_id", userID.String()),
		errors.Token("scheduled_count", persistedCount),
		errors.Token("unscheduled_count", len(schedule.Unscheduled)),
	)

	return nil
}
