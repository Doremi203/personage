package scheduling

import (
	"context"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/Doremi203/personage/backend/libs/go/log"
	"github.com/Doremi203/personage/backend/tasker/internal/usecase/scheduling"
)

type Worker struct {
	useCase *scheduling.UseCase
	logger  log.Logger
}

func NewWorker(useCase *scheduling.UseCase, logger log.Logger) *Worker {
	return &Worker{
		useCase: useCase,
		logger:  logger,
	}
}

func (w *Worker) Process(ctx context.Context) error {
	err := w.useCase.SchedulePendingTasks(ctx)
	if err != nil {
		return errors.WrapFail(err, "schedule pending tasks")
	}

	return nil
}
