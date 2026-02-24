package event

import (
	"context"

	eventsPb "github.com/Doremi203/personage/backend/tasker/gen/api/events"
	"github.com/Doremi203/personage/backend/tasker/internal/domain"
	"github.com/Doremi203/personage/backend/tasker/internal/usecase/clusterization"
)

func NewHandler(clusterizationUseCase *clusterization.UseCase) *handler {
	return &handler{clusterizationUseCase: clusterizationUseCase}
}

type handler struct {
	clusterizationUseCase *clusterization.UseCase
}

func (h *handler) Process(ctx context.Context, data *eventsPb.Event) error {
	event, err := domain.FromPB(data)
	if err != nil {
		return err
	}

	return h.clusterizationUseCase.ProcessEvent(ctx, event)
}
