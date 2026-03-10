package clusterclosure

import (
	"context"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/Doremi203/personage/backend/libs/go/log"
)

type taskGenerationUseCase interface {
	ProcessClosableClusters(ctx context.Context, batchSize int) error
}

type Worker struct {
	useCase   taskGenerationUseCase
	batchSize int
	logger    log.Logger
}

func NewWorker(useCase taskGenerationUseCase, batchSize int, logger log.Logger) *Worker {
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
