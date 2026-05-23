package llm

import (
	"context"

	"github.com/Doremi203/personage/backend/tasker/internal/domain"
)

type PromptProvider interface {
	Get(ctx context.Context, id domain.PromptID) (domain.Prompt, error)
}
