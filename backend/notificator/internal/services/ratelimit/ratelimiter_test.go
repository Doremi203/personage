package ratelimit_test

import (
	"context"
	"testing"

	"github.com/Doremi203/personage/backend/notificator/internal/domain/notification"
	mock_notification "github.com/Doremi203/personage/backend/notificator/internal/domain/notification/mock"
	"github.com/Doremi203/personage/backend/notificator/internal/services/ratelimit"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

var (
	userID = uuid.MustParse("11111111-1111-1111-1111-111111111111")
)

func newLimiter(repo *mock_notification.MockRepo) *ratelimit.RateLimiter {
	return ratelimit.New(repo, map[notification.SettingType]ratelimit.Limits{
		notification.SettingTypeScheduleChange: {Hourly: 2, Daily: 10},
	})
}

func TestRateLimiter_Allow_typeNotInLimits(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mock_notification.NewMockRepo(ctrl)
	// repo should NOT be called for unconstrained types
	limiter := newLimiter(repo)

	allowed, err := limiter.Allow(context.Background(), userID, notification.SettingTypeUpcomingEvent)

	assert.NoError(t, err)
	assert.True(t, allowed)
}

func TestRateLimiter_Allow_withinBothLimits(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mock_notification.NewMockRepo(ctrl)
	limiter := newLimiter(repo)

	repo.EXPECT().
		CountSentSince(gomock.Any(), userID, notification.SettingTypeScheduleChange, gomock.Any()).
		Return(1, nil). // hourly: 1 < 2
		Times(1)
	repo.EXPECT().
		CountSentSince(gomock.Any(), userID, notification.SettingTypeScheduleChange, gomock.Any()).
		Return(5, nil). // daily: 5 < 10
		Times(1)

	allowed, err := limiter.Allow(context.Background(), userID, notification.SettingTypeScheduleChange)

	assert.NoError(t, err)
	assert.True(t, allowed)
}

func TestRateLimiter_Allow_exceedsHourlyLimit(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mock_notification.NewMockRepo(ctrl)
	limiter := newLimiter(repo)

	repo.EXPECT().
		CountSentSince(gomock.Any(), userID, notification.SettingTypeScheduleChange, gomock.Any()).
		Return(2, nil). // hourly: 2 >= 2 → denied
		Times(1)

	allowed, err := limiter.Allow(context.Background(), userID, notification.SettingTypeScheduleChange)

	assert.NoError(t, err)
	assert.False(t, allowed)
}

func TestRateLimiter_Allow_exceedsDailyLimit(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mock_notification.NewMockRepo(ctrl)
	limiter := newLimiter(repo)

	repo.EXPECT().
		CountSentSince(gomock.Any(), userID, notification.SettingTypeScheduleChange, gomock.Any()).
		Return(1, nil). // hourly: 1 < 2
		Times(1)
	repo.EXPECT().
		CountSentSince(gomock.Any(), userID, notification.SettingTypeScheduleChange, gomock.Any()).
		Return(10, nil). // daily: 10 >= 10 → denied
		Times(1)

	allowed, err := limiter.Allow(context.Background(), userID, notification.SettingTypeScheduleChange)

	assert.NoError(t, err)
	assert.False(t, allowed)
}

func TestRateLimiter_Allow_dbError(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mock_notification.NewMockRepo(ctrl)
	limiter := newLimiter(repo)

	repo.EXPECT().
		CountSentSince(gomock.Any(), userID, notification.SettingTypeScheduleChange, gomock.Any()).
		Return(0, assert.AnError)

	allowed, err := limiter.Allow(context.Background(), userID, notification.SettingTypeScheduleChange)

	assert.NoError(t, err)   // no error returned to caller
	assert.False(t, allowed) // fail-safe: deny on DB error
}
