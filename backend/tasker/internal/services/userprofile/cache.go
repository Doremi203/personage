package userprofile

import (
	"context"
	"sync"
	"time"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/Doremi203/personage/backend/tasker/internal/domain"
)

func NewCachedService(
	inner domain.UserProfileService,
	ttl time.Duration,
	negativeTTL time.Duration,
	clock func() time.Time,
) *cachedService {
	return &cachedService{
		inner:       inner,
		ttl:         ttl,
		negativeTTL: negativeTTL,
		clock:       clock,
		entries:     make(map[domain.UserID]cacheEntry),
	}
}

type cacheEntry struct {
	profile   domain.UserProfile
	notFound  bool
	expiresAt time.Time
}

type cachedService struct {
	inner       domain.UserProfileService
	ttl         time.Duration
	negativeTTL time.Duration
	clock       func() time.Time

	mu      sync.Mutex
	entries map[domain.UserID]cacheEntry
}

func (s *cachedService) GetUserProfile(ctx context.Context, userID domain.UserID) (domain.UserProfile, error) {
	now := s.clock()

	s.mu.Lock()
	entry, ok := s.entries[userID]
	if ok && now.Before(entry.expiresAt) {
		s.mu.Unlock()
		if entry.notFound {
			return domain.UserProfile{}, domain.ErrUserProfileNotFound
		}
		return entry.profile, nil
	}
	s.mu.Unlock()

	profile, err := s.inner.GetUserProfile(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrUserProfileNotFound) {
			s.mu.Lock()
			s.entries[userID] = cacheEntry{
				notFound:  true,
				expiresAt: now.Add(s.negativeTTL),
			}
			s.mu.Unlock()
		}
		return domain.UserProfile{}, err
	}

	s.mu.Lock()
	s.entries[userID] = cacheEntry{
		profile:   profile,
		expiresAt: now.Add(s.ttl),
	}
	s.mu.Unlock()

	return profile, nil
}
