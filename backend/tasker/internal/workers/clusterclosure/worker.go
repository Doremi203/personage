package clusterclosure

import (
	"context"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/Doremi203/personage/backend/libs/go/log"
	"github.com/Doremi203/personage/backend/tasker/internal/usecase/taskgeneration"
)

type Worker struct {
	useCase *taskgeneration.UseCase
	logger  log.Logger
}

func NewWorker(useCase *taskgeneration.UseCase, logger log.Logger) *Worker {
	return &Worker{
		useCase: useCase,
		logger:  logger,
	}
}

func (w *Worker) Process(ctx context.Context) error {
	err := w.useCase.ProcessClosableClusters(ctx)
	if err != nil {
		return errors.WrapFail(err, "process closable clusters")
	}

	return nil
}
