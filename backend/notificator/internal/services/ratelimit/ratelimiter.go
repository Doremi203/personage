package ratelimit

import (
	"context"
	"time"

	"github.com/Doremi203/personage/backend/notificator/internal/domain/notification"
	"github.com/google/uuid"
)

type Limits struct {
	Hourly int
	Daily  int
}

type RateLimiter struct {
	repo   notification.Repo
	limits map[notification.SettingType]Limits
}

func New(repo notification.Repo, limits map[notification.SettingType]Limits) *RateLimiter {
	return &RateLimiter{repo: repo, limits: limits}
}

func (r *RateLimiter) Allow(ctx context.Context, userID uuid.UUID, typ notification.SettingType) (bool, error) {
	limits, ok := r.limits[typ]
	if !ok {
		return true, nil
	}

	now := time.Now()

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
