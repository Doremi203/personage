package prompts

import (
	"context"
	"sync"
	"time"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/Doremi203/personage/backend/tasker/internal/domain"
)

func NewService(repo domain.PromptRepo, ttl time.Duration, clock func() time.Time) *Service {
	return &Service{
		repo:  repo,
		ttl:   ttl,
		clock: clock,
		cache: make(map[domain.PromptID]cachedPrompt),
	}
}

type Service struct {
	repo  domain.PromptRepo
	ttl   time.Duration
	clock func() time.Time

	mu    sync.RWMutex
	cache map[domain.PromptID]cachedPrompt
}

type cachedPrompt struct {
	value     domain.Prompt
	expiresAt time.Time
}

func (s *Service) Get(ctx context.Context, id domain.PromptID) (domain.Prompt, error) {
	if p, ok := s.lookup(id); ok {
		return p, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if entry, ok := s.cache[id]; ok && s.clock().Before(entry.expiresAt) {
		return entry.value, nil
	}

	prompt, err := s.repo.GetPrompt(ctx, id)
	if err != nil {
		return domain.Prompt{}, errors.WrapFailf(err, "load prompt %s", errors.Token("prompt_id", id.String()))
	}

	s.cache[id] = cachedPrompt{
		value:     prompt,
		expiresAt: s.clock().Add(s.ttl),
	}
	return prompt, nil
}

func (s *Service) Invalidate(id domain.PromptID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.cache, id)
}

func (s *Service) lookup(id domain.PromptID) (domain.Prompt, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, ok := s.cache[id]
	if !ok {
		return domain.Prompt{}, false
	}
	if !s.clock().Before(entry.expiresAt) {
		return domain.Prompt{}, false
	}
	return entry.value, true
}
