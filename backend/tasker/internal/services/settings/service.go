package settings

import (
	"context"
	"sync"
	"time"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/Doremi203/personage/backend/libs/go/log"
	"github.com/Doremi203/personage/backend/tasker/internal/domain"
)

func NewService(
	repo domain.GenerationSettingsRepo,
	ttl time.Duration,
	clock func() time.Time,
	defaults domain.GenerationSettings,
	logger log.Logger,
) *Service {
	return &Service{
		repo:     repo,
		ttl:      ttl,
		clock:    clock,
		defaults: defaults,
		logger:   logger,
	}
}

type Service struct {
	repo     domain.GenerationSettingsRepo
	ttl      time.Duration
	clock    func() time.Time
	defaults domain.GenerationSettings
	logger   log.Logger

	mu        sync.RWMutex
	cached    domain.GenerationSettings
	hasCached bool
	expiresAt time.Time
}

func (s *Service) GenerationSettings(ctx context.Context) (domain.GenerationSettings, error) {
	if value, ok := s.lookup(); ok {
		return value, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.hasCached && s.clock().Before(s.expiresAt) {
		return s.cached, nil
	}

	value, err := s.repo.GetGenerationSettings(ctx)
	if err != nil {
		if s.hasCached {
			// Back off for a full TTL so an outage does not make every call hit the DB.
			s.expiresAt = s.clock().Add(s.ttl)
			s.logger.Warn(errors.WrapFail(err, "refresh generation settings, serving stale cached value"))
			return s.cached, nil
		}
		s.logger.Error(errors.WrapFail(err, "load generation settings, using defaults"))
		return s.defaults, nil
	}

	s.cached = value
	s.hasCached = true
	s.expiresAt = s.clock().Add(s.ttl)
	return value, nil
}

func (s *Service) Invalidate() {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Expire immediately but keep the cached value so a failed refresh can still
	// fall back to the last known settings instead of injected defaults.
	s.expiresAt = time.Time{}
}

func (s *Service) lookup() (domain.GenerationSettings, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.hasCached {
		return domain.GenerationSettings{}, false
	}
	if !s.clock().Before(s.expiresAt) {
		return domain.GenerationSettings{}, false
	}
	return s.cached, true
}
