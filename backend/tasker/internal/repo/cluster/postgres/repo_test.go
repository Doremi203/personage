package clusterpostgres

import (
	"context"
	"testing"
	"time"

	"github.com/Doremi203/personage/backend/libs/go/postgres"
	"github.com/Doremi203/personage/backend/tasker/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const embeddingDim = 1536

func makeEmbedding(seed float32) []float32 {
	v := make([]float32, embeddingDim)
	v[0] = seed
	return v
}

func newCluster(t *testing.T, userID domain.UserID, status domain.ClusterStatus, eventCount int, updatedAt time.Time) domain.Cluster {
	t.Helper()
	return domain.Cluster{
		ID:         domain.ClusterID(uuid.NewString()),
		UserID:     userID,
		Centroid:   makeEmbedding(1),
		EventCount: eventCount,
		Status:     status,
		CreatedAt:  updatedAt,
		UpdatedAt:  updatedAt,
	}
}

func fixedClock(t time.Time) func() time.Time {
	return func() time.Time { return t }
}

func Test_repo_UpsertCluster(t *testing.T) {
	userA := domain.UserID(uuid.NewString())

	tester.Run(t, "insert then update", nil, 10*time.Second,
		func(t *testing.T, ctx context.Context, db postgres.Client) {
			now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
			r := NewRepo(db, fixedClock(now))

			c := newCluster(t, userA, domain.ClusterStatusOpen, 1, now)
			require.NoError(t, r.UpsertCluster(ctx, c))

			c.EventCount = 5
			c.Status = domain.ClusterStatusProcessing
			c.UpdatedAt = now.Add(time.Minute)
			require.NoError(t, r.UpsertCluster(ctx, c))

			diags, err := r.ListGenerationDiagnosticsByUserID(ctx, userA)
			require.NoError(t, err)
			require.Len(t, diags, 1)
			assert.Equal(t, 5, diags[0].EventCount)
			assert.Equal(t, domain.ClusterStatusProcessing, diags[0].Status)
		},
	)
}

func Test_repo_FindSimilarClusters(t *testing.T) {
	userA := domain.UserID(uuid.NewString())
	userB := domain.UserID(uuid.NewString())

	tester.Run(t, "returns top-k for same user, only open", nil, 10*time.Second,
		func(t *testing.T, ctx context.Context, db postgres.Client) {
			now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
			r := NewRepo(db, fixedClock(now))

			open1 := newCluster(t, userA, domain.ClusterStatusOpen, 1, now)
			open1.Centroid = makeEmbedding(1)
			open2 := newCluster(t, userA, domain.ClusterStatusOpen, 1, now)
			open2.Centroid = makeEmbedding(0.5)
			closed := newCluster(t, userA, domain.ClusterStatusClosed, 1, now)
			closed.Centroid = makeEmbedding(1)
			otherUser := newCluster(t, userB, domain.ClusterStatusOpen, 1, now)
			otherUser.Centroid = makeEmbedding(1)

			require.NoError(t, r.UpsertCluster(ctx, open1))
			require.NoError(t, r.UpsertCluster(ctx, open2))
			require.NoError(t, r.UpsertCluster(ctx, closed))
			require.NoError(t, r.UpsertCluster(ctx, otherUser))

			got, err := r.FindSimilarClusters(ctx, userA, makeEmbedding(1), 5)
			require.NoError(t, err)
			require.Len(t, got, 2)
			assert.Equal(t, open1.ID, got[0].ID)
		},
	)
}

func Test_repo_FindClosableClusters(t *testing.T) {
	userA := domain.UserID(uuid.NewString())

	tester.Run(t, "selects by event count or inactivity", nil, 10*time.Second,
		func(t *testing.T, ctx context.Context, db postgres.Client) {
			now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
			r := NewRepo(db, fixedClock(now))

			ready := newCluster(t, userA, domain.ClusterStatusOpen, 10, now)
			stale := newCluster(t, userA, domain.ClusterStatusOpen, 1, now.Add(-time.Hour))
			fresh := newCluster(t, userA, domain.ClusterStatusOpen, 1, now)
			closed := newCluster(t, userA, domain.ClusterStatusClosed, 100, now)
			require.NoError(t, r.UpsertCluster(ctx, ready))
			require.NoError(t, r.UpsertCluster(ctx, stale))
			require.NoError(t, r.UpsertCluster(ctx, fresh))
			require.NoError(t, r.UpsertCluster(ctx, closed))

			got, err := r.FindClosableClusters(ctx, 5, 30*time.Minute, 10)
			require.NoError(t, err)
			require.Len(t, got, 2)
			ids := []domain.ClusterID{got[0].ID, got[1].ID}
			assert.ElementsMatch(t, []domain.ClusterID{ready.ID, stale.ID}, ids)
		},
	)
}

func Test_repo_UpdateClusterStatus(t *testing.T) {
	userA := domain.UserID(uuid.NewString())

	tester.Run(t, "updates status and updated_at", nil, 10*time.Second,
		func(t *testing.T, ctx context.Context, db postgres.Client) {
			created := time.Date(2026, 4, 26, 10, 0, 0, 0, time.UTC)
			now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
			r := NewRepo(db, fixedClock(now))

			c := newCluster(t, userA, domain.ClusterStatusOpen, 1, created)
			require.NoError(t, r.UpsertCluster(ctx, c))

			require.NoError(t, r.UpdateClusterStatus(ctx, c.ID, domain.ClusterStatusProcessing))

			diags, err := r.ListGenerationDiagnosticsByUserID(ctx, userA)
			require.NoError(t, err)
			require.Len(t, diags, 1)
			assert.Equal(t, domain.ClusterStatusProcessing, diags[0].Status)
			assert.True(t, diags[0].UpdatedAt.Equal(now))
		},
	)
}

func Test_repo_FinalizeCluster(t *testing.T) {
	userA := domain.UserID(uuid.NewString())

	tester.Run(t, "marks closed with outcome and reason", nil, 10*time.Second,
		func(t *testing.T, ctx context.Context, db postgres.Client) {
			now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
			r := NewRepo(db, fixedClock(now))

			c := newCluster(t, userA, domain.ClusterStatusProcessing, 1, now)
			require.NoError(t, r.UpsertCluster(ctx, c))

			reason := "no signal"
			require.NoError(t, r.FinalizeCluster(ctx, c.ID, domain.ClusterGenerationOutcomeNonActionable, &reason))

			diags, err := r.ListGenerationDiagnosticsByUserID(ctx, userA)
			require.NoError(t, err)
			require.Len(t, diags, 1)
			assert.Equal(t, domain.ClusterStatusClosed, diags[0].Status)
			require.NotNil(t, diags[0].GenerationOutcome)
			assert.Equal(t, domain.ClusterGenerationOutcomeNonActionable, *diags[0].GenerationOutcome)
			require.NotNil(t, diags[0].GenerationReason)
			assert.Equal(t, reason, *diags[0].GenerationReason)
		},
	)
}

func Test_repo_DeleteCluster(t *testing.T) {
	userA := domain.UserID(uuid.NewString())

	tester.Run(t, "deletes existing cluster", nil, 10*time.Second,
		func(t *testing.T, ctx context.Context, db postgres.Client) {
			now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
			r := NewRepo(db, fixedClock(now))

			c := newCluster(t, userA, domain.ClusterStatusOpen, 1, now)
			require.NoError(t, r.UpsertCluster(ctx, c))

			require.NoError(t, r.DeleteCluster(ctx, c.ID))

			diags, err := r.ListGenerationDiagnosticsByUserID(ctx, userA)
			require.NoError(t, err)
			assert.Empty(t, diags)
		},
	)

	tester.Run(t, "unknown cluster returns error", nil, 10*time.Second,
		func(t *testing.T, ctx context.Context, db postgres.Client) {
			now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
			r := NewRepo(db, fixedClock(now))

			err := r.DeleteCluster(ctx, domain.ClusterID(uuid.NewString()))
			require.Error(t, err)
		},
	)
}

func Test_repo_RecoverStaleClusters(t *testing.T) {
	userA := domain.UserID(uuid.NewString())

	tester.Run(t, "moves stale processing clusters back to open", nil, 10*time.Second,
		func(t *testing.T, ctx context.Context, db postgres.Client) {
			now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
			r := NewRepo(db, fixedClock(now))

			stale := newCluster(t, userA, domain.ClusterStatusProcessing, 1, now.Add(-time.Hour))
			fresh := newCluster(t, userA, domain.ClusterStatusProcessing, 1, now)
			open := newCluster(t, userA, domain.ClusterStatusOpen, 1, now.Add(-time.Hour))
			require.NoError(t, r.UpsertCluster(ctx, stale))
			require.NoError(t, r.UpsertCluster(ctx, fresh))
			require.NoError(t, r.UpsertCluster(ctx, open))

			n, err := r.RecoverStaleClusters(ctx, 30*time.Minute)
			require.NoError(t, err)
			assert.Equal(t, 1, n)

			diags, err := r.ListGenerationDiagnosticsByUserID(ctx, userA)
			require.NoError(t, err)
			byID := map[domain.ClusterID]domain.ClusterStatus{}
			for _, d := range diags {
				byID[d.ClusterID] = d.Status
			}
			assert.Equal(t, domain.ClusterStatusOpen, byID[stale.ID])
			assert.Equal(t, domain.ClusterStatusProcessing, byID[fresh.ID])
			assert.Equal(t, domain.ClusterStatusOpen, byID[open.ID])
		},
	)
}

func Test_repo_ListGenerationDiagnosticsByUserID(t *testing.T) {
	userA := domain.UserID(uuid.NewString())
	userB := domain.UserID(uuid.NewString())

	tester.Run(t, "returns only requested user", nil, 10*time.Second,
		func(t *testing.T, ctx context.Context, db postgres.Client) {
			now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
			r := NewRepo(db, fixedClock(now))

			require.NoError(t, r.UpsertCluster(ctx, newCluster(t, userA, domain.ClusterStatusOpen, 1, now)))
			require.NoError(t, r.UpsertCluster(ctx, newCluster(t, userB, domain.ClusterStatusOpen, 1, now)))

			diags, err := r.ListGenerationDiagnosticsByUserID(ctx, userA)
			require.NoError(t, err)
			require.Len(t, diags, 1)
			assert.Equal(t, userA, diags[0].UserID)
			assert.Equal(t, 0, diags[0].GeneratedTaskCount)
		},
	)
}
