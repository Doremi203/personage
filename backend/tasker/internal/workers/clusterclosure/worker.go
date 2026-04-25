package clusterclosure

import (
	"context"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/Doremi203/personage/backend/libs/go/log"
	"github.com/Doremi203/personage/backend/tasker/internal/usecase/taskgeneration"
)

type Worker struct {
	useCase   *taskgeneration.UseCase
	batchSize int
	logger    log.Logger
}

func NewWorker(useCase *taskgeneration.UseCase, batchSize int, logger log.Logger) *Worker {
	return &Worker{
		useCase:   useCase,
		batchSize: batchSize,
		logger:    logger,
	}
}

func (w *Worker) Process(ctx context.Context) error {
	err := w.useCase.ProcessClosableClusters(ctx, w.batchSize)
	if err != nil {
		return errors.WrapFail(err, "process closable clusters")
	}

	return nil
}
