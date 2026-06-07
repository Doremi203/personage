package generationsettingspostgres

import (
	"context"
	"testing"
	"time"

	"github.com/Doremi203/personage/backend/libs/go/postgres"
	"github.com/Doremi203/personage/backend/tasker/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_repo_GetGenerationSettings_ReturnsSeededDefaults(t *testing.T) {
	tester.Run(t, "get returns seeded singleton row", nil, 10*time.Second,
		func(t *testing.T, ctx context.Context, db postgres.Client) {
			r := NewRepo(db, time.Now)

			got, err := r.GetGenerationSettings(ctx)
			require.NoError(t, err)

			assert.InDelta(t, 0.65, got.MinSimilarity, 1e-9)
			assert.InDelta(t, 0.90, got.ClosedSimilarityThreshold, 1e-9)
			assert.Equal(t, 5, got.TopK)
			assert.Equal(t, 5, got.MaxEventCount)
			assert.Equal(t, 5*time.Minute, got.InactivityTimeout)
			assert.Equal(t, 10, got.BatchSize)
			assert.InDelta(t, 0.97, got.TaskDuplicateThreshold, 1e-9)
		},
	)
}

func Test_repo_UpdateGenerationSettings_PartialCoalesce(t *testing.T) {
	fixedNow := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)

	tester.Run(t, "partial update keeps untouched fields and maps minutes to duration", nil, 10*time.Second,
		func(t *testing.T, ctx context.Context, db postgres.Client) {
			r := NewRepo(db, func() time.Time { return fixedNow })

			newMinSimilarity := 0.42
			newInactivityMinutes := 12
			newTaskDuplicateThreshold := 0.85

			updated, err := r.UpdateGenerationSettings(ctx, domain.GenerationSettingsUpdate{
				MinSimilarity:          &newMinSimilarity,
				InactivityMinutes:      &newInactivityMinutes,
				TaskDuplicateThreshold: &newTaskDuplicateThreshold,
			})
			require.NoError(t, err)

			assert.InDelta(t, newMinSimilarity, updated.MinSimilarity, 1e-9)
			assert.Equal(t, 12*time.Minute, updated.InactivityTimeout)
			assert.InDelta(t, newTaskDuplicateThreshold, updated.TaskDuplicateThreshold, 1e-9)
			// untouched fields keep their seeded defaults
			assert.InDelta(t, 0.90, updated.ClosedSimilarityThreshold, 1e-9)
			assert.Equal(t, 5, updated.TopK)
			assert.Equal(t, 5, updated.MaxEventCount)
			assert.Equal(t, 10, updated.BatchSize)
			assert.Equal(t, fixedNow.UTC(), updated.UpdatedAt.UTC())

			got, err := r.GetGenerationSettings(ctx)
			require.NoError(t, err)
			assert.InDelta(t, newMinSimilarity, got.MinSimilarity, 1e-9)
			assert.Equal(t, 12*time.Minute, got.InactivityTimeout)
			assert.InDelta(t, newTaskDuplicateThreshold, got.TaskDuplicateThreshold, 1e-9)
		},
	)
}
