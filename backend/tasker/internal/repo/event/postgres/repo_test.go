package eventpostgres

import (
	"context"
	"testing"
	"time"

	"github.com/Doremi203/personage/backend/libs/go/postgres"
	"github.com/Doremi203/personage/backend/tasker/internal/domain"
	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const embeddingDim = 1536

func makeEmbedding(seed float32) []float32 {
	v := make([]float32, embeddingDim)
	v[0] = seed
	return v
}

func insertCluster(t *testing.T, ctx context.Context, db postgres.Client, userID domain.UserID, now time.Time) domain.ClusterID {
	t.Helper()
	id := domain.ClusterID(uuid.NewString())
	_, err := db.Exec(ctx, `
		INSERT INTO clusters (cluster_id, user_id, centroid, event_count, status, created_at, updated_at)
		VALUES ($1, $2, $3, 0, 'open', $4, $4)
	`, id, userID, pgvector.NewVector(makeEmbedding(1)), now)
	require.NoError(t, err)
	return id
}

func newEvent(userID domain.UserID, clusterID domain.ClusterID) domain.EventWithEmbedding {
	return domain.EventWithEmbedding{
		Event: domain.Event{
			ID:         domain.EventID(uuid.NewString()),
			UserID:     userID,
			Source:     domain.EventSourceGmail,
			Context:    domain.NormalizedEventContext(`{"k":"v"}`),
			OccurredAt: time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC),
			ClusterID:  clusterID,
		},
		Embedding: makeEmbedding(0.5),
	}
}

func Test_repo_UpsertEvent(t *testing.T) {
	userA := domain.UserID(uuid.NewString())

	tester.Run(t, "insert then update", nil, 10*time.Second,
		func(t *testing.T, ctx context.Context, db postgres.Client) {
			now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
			r := NewRepo(db, func() time.Time { return now })

			clusterID := insertCluster(t, ctx, db, userA, now)
			ev := newEvent(userA, clusterID)
			require.NoError(t, r.UpsertEvent(ctx, ev))

			ev.Source = domain.EventSourceTelegram
			require.NoError(t, r.UpsertEvent(ctx, ev))

			got, err := r.GetEventsByClusterID(ctx, clusterID)
			require.NoError(t, err)
			require.Len(t, got, 1)
			assert.Equal(t, domain.EventSourceTelegram, got[0].Source)
		},
	)
}

func Test_repo_GetEventsByClusterID(t *testing.T) {
	userA := domain.UserID(uuid.NewString())

	tester.Run(t, "ordered by occurred_at asc", nil, 10*time.Second,
		func(t *testing.T, ctx context.Context, db postgres.Client) {
			now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
			r := NewRepo(db, func() time.Time { return now })

			clusterID := insertCluster(t, ctx, db, userA, now)

			e1 := newEvent(userA, clusterID)
			e1.OccurredAt = time.Date(2026, 4, 26, 14, 0, 0, 0, time.UTC)
			e2 := newEvent(userA, clusterID)
			e2.OccurredAt = time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
			require.NoError(t, r.UpsertEvent(ctx, e1))
			require.NoError(t, r.UpsertEvent(ctx, e2))

			got, err := r.GetEventsByClusterID(ctx, clusterID)
			require.NoError(t, err)
			require.Len(t, got, 2)
			assert.Equal(t, e2.ID, got[0].ID)
			assert.Equal(t, e1.ID, got[1].ID)
		},
	)

	tester.Run(t, "no events returns empty", nil, 10*time.Second,
		func(t *testing.T, ctx context.Context, db postgres.Client) {
			now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
			r := NewRepo(db, func() time.Time { return now })

			got, err := r.GetEventsByClusterID(ctx, domain.ClusterID(uuid.NewString()))
			require.NoError(t, err)
			assert.Empty(t, got)
		},
	)
}

func Test_repo_DeleteEventsByClusterID(t *testing.T) {
	userA := domain.UserID(uuid.NewString())

	tester.Run(t, "removes events of given cluster only", nil, 10*time.Second,
		func(t *testing.T, ctx context.Context, db postgres.Client) {
			now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
			r := NewRepo(db, func() time.Time { return now })

			clusterA := insertCluster(t, ctx, db, userA, now)
			clusterB := insertCluster(t, ctx, db, userA, now)

			require.NoError(t, r.UpsertEvent(ctx, newEvent(userA, clusterA)))
			require.NoError(t, r.UpsertEvent(ctx, newEvent(userA, clusterA)))
			require.NoError(t, r.UpsertEvent(ctx, newEvent(userA, clusterB)))

			require.NoError(t, r.DeleteEventsByClusterID(ctx, clusterA))

			leftA, err := r.GetEventsByClusterID(ctx, clusterA)
			require.NoError(t, err)
			assert.Empty(t, leftA)

			leftB, err := r.GetEventsByClusterID(ctx, clusterB)
			require.NoError(t, err)
			assert.Len(t, leftB, 1)
		},
	)
}
