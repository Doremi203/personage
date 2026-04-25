package ratelimit

import (
	"context"
	"time"

	"github.com/Doremi203/personage/backend/notificator/internal/domain/notification"
	"github.com/google/uuid"
)

//go:generate mockgen -source=ratelimiter.go -destination=mock/ratelimiter_mock.go -typed

type Allower interface {
	Allow(ctx context.Context, userID uuid.UUID, typ notification.SettingType) (bool, error)
}

type Limits struct {
	Hourly int
	Daily  int
}

type RateLimiter struct {
	repo   notification.Repo
	limits map[notification.SettingType]Limits
	clock  func() time.Time
}

func New(repo notification.Repo, limits map[notification.SettingType]Limits, clock func() time.Time) *RateLimiter {
	return &RateLimiter{repo: repo, limits: limits, clock: clock}
}

func (r *RateLimiter) Allow(ctx context.Context, userID uuid.UUID, typ notification.SettingType) (bool, error) {
	limits, ok := r.limits[typ]
	if !ok {
		return true, nil
	}

	now := r.clock()

	hourlyCount, err := r.repo.CountSentSince(ctx, userID, typ, now.Add(-time.Hour))
	if err != nil {
		return false, nil // fail-safe: deny on DB error
	}
	if hourlyCount >= limits.Hourly {
		return false, nil
	}

	dailyCount, err := r.repo.CountSentSince(ctx, userID, typ, now.Add(-24*time.Hour))
	if err != nil {
		return false, nil
	}
	if dailyCount >= limits.Daily {
		return false, nil
	}

	return true, nil
}
