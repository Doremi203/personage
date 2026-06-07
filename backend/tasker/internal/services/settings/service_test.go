package settings_test

import (
	"testing"
	"time"

	"github.com/Doremi203/personage/backend/libs/go/log"
	"github.com/Doremi203/personage/backend/tasker/internal/domain"
	mock_domain "github.com/Doremi203/personage/backend/tasker/internal/domain/mock"
	"github.com/Doremi203/personage/backend/tasker/internal/services/settings"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const testTTL = 30 * time.Second

func newClock(t *time.Time) func() time.Time {
	return func() time.Time { return *t }
}

func TestService_CachesWithinTTL(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mock_domain.NewMockGenerationSettingsRepo(ctrl)
	now := time.Unix(0, 0).UTC()
	stored := domain.GenerationSettings{MinSimilarity: 0.7, TopK: 5}

	repo.EXPECT().GetGenerationSettings(gomock.Any()).Return(stored, nil).Times(1)

	s := settings.NewService(repo, testTTL, newClock(&now), domain.GenerationSettings{}, log.Stub{})

	first, err := s.GenerationSettings(t.Context())
	require.NoError(t, err)
	assert.Equal(t, stored, first)

	now = now.Add(testTTL - time.Second)
	second, err := s.GenerationSettings(t.Context())
	require.NoError(t, err)
	assert.Equal(t, stored, second)
}

func TestService_StaleFallbackBacksOff(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mock_domain.NewMockGenerationSettingsRepo(ctrl)
	now := time.Unix(0, 0).UTC()
	stored := domain.GenerationSettings{MinSimilarity: 0.7, TopK: 5}

	gomock.InOrder(
		repo.EXPECT().GetGenerationSettings(gomock.Any()).Return(stored, nil),
		repo.EXPECT().GetGenerationSettings(gomock.Any()).Return(domain.GenerationSettings{}, assert.AnError),
	)

	s := settings.NewService(repo, testTTL, newClock(&now), domain.GenerationSettings{MinSimilarity: 0.1}, log.Stub{})

	_, err := s.GenerationSettings(t.Context())
	require.NoError(t, err)

	// TTL expired: refresh fails but the last value is served and the TTL is extended.
	now = now.Add(testTTL)
	stale, err := s.GenerationSettings(t.Context())
	require.NoError(t, err)
	assert.Equal(t, stored, stale)

	// Within the new TTL window the repo is not hit again (no extra EXPECT registered).
	now = now.Add(testTTL - time.Second)
	again, err := s.GenerationSettings(t.Context())
	require.NoError(t, err)
	assert.Equal(t, stored, again)
}

func TestService_InvalidateKeepsCachedOnRefreshFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mock_domain.NewMockGenerationSettingsRepo(ctrl)
	now := time.Unix(0, 0).UTC()
	stored := domain.GenerationSettings{MinSimilarity: 0.7, TopK: 5}
	defaults := domain.GenerationSettings{MinSimilarity: 0.1}

	gomock.InOrder(
		repo.EXPECT().GetGenerationSettings(gomock.Any()).Return(stored, nil),
		repo.EXPECT().GetGenerationSettings(gomock.Any()).Return(domain.GenerationSettings{}, assert.AnError),
	)

	s := settings.NewService(repo, testTTL, newClock(&now), defaults, log.Stub{})

	_, err := s.GenerationSettings(t.Context())
	require.NoError(t, err)

	s.Invalidate()

	// Invalidate forces a refresh; it fails, so the last cached value must be served, not defaults.
	value, err := s.GenerationSettings(t.Context())
	require.NoError(t, err)
	assert.Equal(t, stored, value)
}

func TestService_DefaultsWhenNeverCached(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mock_domain.NewMockGenerationSettingsRepo(ctrl)
	now := time.Unix(0, 0).UTC()
	defaults := domain.GenerationSettings{MinSimilarity: 0.42, TopK: 3}

	repo.EXPECT().GetGenerationSettings(gomock.Any()).Return(domain.GenerationSettings{}, assert.AnError)

	s := settings.NewService(repo, testTTL, newClock(&now), defaults, log.Stub{})

	value, err := s.GenerationSettings(t.Context())
	require.NoError(t, err)
	assert.Equal(t, defaults, value)
}
