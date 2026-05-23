package event

import (
	"context"
	"time"

	eventsPb "github.com/Doremi203/personage/backend/tasker/gen/api/events"
	"github.com/Doremi203/personage/backend/tasker/internal/domain"
	"github.com/Doremi203/personage/backend/tasker/internal/usecase/clusterization"
)

func NewHandler(clusterizationUseCase *clusterization.UseCase, defaultLocation *time.Location) *handler {
	return &handler{
		clusterizationUseCase: clusterizationUseCase,
		defaultLocation:       defaultLocation,
	}
}

type handler struct {
	clusterizationUseCase *clusterization.UseCase
	defaultLocation       *time.Location
}

func (h *handler) Process(ctx context.Context, data *eventsPb.Event) error {
	event, err := domain.FromPB(data, h.defaultLocation)
	if err != nil {
		return err
	}

	return h.clusterizationUseCase.ProcessEvent(ctx, event)
}
