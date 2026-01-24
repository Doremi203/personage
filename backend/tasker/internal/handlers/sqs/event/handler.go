package event

import (
	"context"

	eventsPb "gitlab.com/amoguscorp/personage/backend/tasker/gen/api/events"
	"gitlab.com/amoguscorp/personage/backend/tasker/internal/domain"
	"gitlab.com/amoguscorp/personage/backend/tasker/internal/usecase/clusterization"
)

func NewHandler(clusterizationUseCase *clusterization.UseCase) *handler {
	return &handler{clusterizationUseCase: clusterizationUseCase}
}

type handler struct {
	clusterizationUseCase *clusterization.UseCase
}

func (h *handler) Process(ctx context.Context, data *eventsPb.Event) error {
	return h.clusterizationUseCase.ProcessEvent(ctx, domain.FromPB(data))
}
