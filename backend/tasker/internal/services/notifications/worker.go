package notifications

import (
	"context"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/Doremi203/personage/backend/libs/go/log"
	"github.com/Doremi203/personage/backend/tasker/internal/domain"
)

func NewWorker(
	logger log.Logger,
	taskRepo domain.TaskRepo,
	upcomingEventNotifier domain.UpcomingEventNotifier,
	scheduleChangeNotifier domain.ScheduleChangeNotifier,
) *Worker {
	return &Worker{
		logger:                 logger,
		taskRepo:               taskRepo,
		upcomingEventNotifier:  upcomingEventNotifier,
		scheduleChangeNotifier: scheduleChangeNotifier,
	}
}

type Worker struct {
	upcomingEventNotifier  domain.UpcomingEventNotifier
	scheduleChangeNotifier domain.ScheduleChangeNotifier
	taskRepo               domain.TaskRepo
	logger                 log.Logger
}

func (w *Worker) Process(ctx context.Context) error {
	userIDs, err := w.taskRepo.GetUsersWithUnplannedTasks(ctx)
	if err != nil {
		return errors.WrapFail(err, "get users for notification check")
	}

	if len(userIDs) == 0 {
		return nil
	}

	for _, userID := range userIDs {
		if err := w.processUser(ctx, userID); err != nil {
			w.logger.Error(errors.WrapFailf(
				err,
				"process notifications for user %s",
				errors.Token("user_id", userID.String()),
			))
			continue
		}
	}

	return nil
}

func (w *Worker) processUser(ctx context.Context, userID domain.UserID) error {
	plannedTasks, err := w.taskRepo.GetTasksByStatus(ctx, userID, domain.TaskStatusPlanned)
	if err != nil {
		return errors.WrapFail(err, "get planned tasks")
	}

	if len(plannedTasks) == 0 {
		return nil
	}

	if err := w.upcomingEventNotifier.NotifyUpcomingEvents(ctx, userID, plannedTasks); err != nil {
		return errors.WrapFail(err, "notify upcoming events")
	}

	return nil
}
