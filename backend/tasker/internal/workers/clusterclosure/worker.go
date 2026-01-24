package clusterclosure

import (
	"context"

	"gitlab.com/amoguscorp/personage/backend/libs/go/errors"
	"gitlab.com/amoguscorp/personage/backend/libs/go/log"
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
	w.logger.Infof("starting cluster closure processing")

	err := w.useCase.ProcessClosableClusters(ctx, w.batchSize)
	if err != nil {
		return errors.WrapFail(err, "process closable clusters")
	}

	w.logger.Infof("cluster closure processing completed")
	return nil
}
