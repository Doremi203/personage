package scheduling

import (
	"context"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/Doremi203/personage/backend/libs/go/log"
)

type schedulingUseCase interface {
	SchedulePendingTasks(ctx context.Context) error
}

type Worker struct {
	useCase schedulingUseCase
	logger  log.Logger
}

func NewWorker(useCase schedulingUseCase, logger log.Logger) *Worker {
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
