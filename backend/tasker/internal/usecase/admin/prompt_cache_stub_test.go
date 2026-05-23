package admin_test

import "github.com/Doremi203/personage/backend/tasker/internal/domain"

type noopPromptCache struct{}

func (noopPromptCache) Invalidate(domain.PromptID) {}
