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
) *Worker {
	return &Worker{
		logger:                logger,
		taskRepo:              taskRepo,
		upcomingEventNotifier: upcomingEventNotifier,
	}
}

type Worker struct {
	upcomingEventNotifier domain.UpcomingEventNotifier
	taskRepo              domain.TaskRepo
	logger                log.Logger
}

func (w *Worker) Process(ctx context.Context) error {
	userIDs, err := w.taskRepo.GetUsersWithPlannedTasks(ctx)
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
		w.logger.Infof(
			"no planned tasks for user %s",
			errors.Token("user_id", userID.String()),
		)
		return nil
	}

	w.logger.Infof(
		"checking upcoming events for user %s %s",
		errors.Token("user_id", userID.String()),
		errors.Token("planned_count", len(plannedTasks)),
	)

	if err := w.upcomingEventNotifier.NotifyUpcomingEvents(ctx, userID, plannedTasks); err != nil {
		return errors.WrapFail(err, "notify upcoming events")
	}

	w.logger.Infof(
		"upcoming event check completed for user %s",
		errors.Token("user_id", userID.String()),
	)

	return nil
}
