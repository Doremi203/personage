package eventpostgres

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/Doremi203/personage/backend/libs/go/postgres"
	"github.com/Doremi203/personage/backend/libs/go/slices"
	"github.com/Doremi203/personage/backend/tasker/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pgvector/pgvector-go"
)

func NewRepo(client postgres.Client, clock func() time.Time) *repo {
	return &repo{
		client: client,
		clock:  clock,
	}
}

type repo struct {
	client postgres.Client
	clock  func() time.Time
}

func (r *repo) UpsertEvent(ctx context.Context, event domain.EventWithEmbedding) error {
	query := `
		INSERT INTO events (
			event_id,
		    user_id,
		    source,
		    occurred_at,
		    context,
		    embedding,
		    cluster_id,
		    similarity,
			created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)
		ON CONFLICT (event_id) DO UPDATE SET
			source = EXCLUDED.source,
		    user_id = EXCLUDED.user_id,
			occurred_at = EXCLUDED.occurred_at,
			context = EXCLUDED.context,
			embedding = EXCLUDED.embedding,
			cluster_id = EXCLUDED.cluster_id,
			similarity = EXCLUDED.similarity,
			created_at = EXCLUDED.created_at
	`

	_, err := r.client.Exec(ctx, query,
		event.ID,
		event.UserID,
		event.Source,
		event.OccurredAt,
		event.Context,
		pgvector.NewVector(event.Embedding),
		event.ClusterID,
		event.Similarity,
		r.clock(),
	)

	if err != nil {
		return errors.WrapFail(err, "upsert event")
	}

	return nil
}

func (r *repo) GetEventsByClusterID(ctx context.Context, clusterID domain.ClusterID) ([]domain.Event, error) {
	query := `
		SELECT
			event_id,
			user_id,
			source,
			occurred_at,
			context,
			cluster_id,
			similarity
		FROM events
		WHERE cluster_id = $1
		ORDER BY occurred_at ASC
	`

	rows, err := r.client.Query(ctx, query, clusterID)
	if err != nil {
		return nil, errors.WrapFail(err, "get events by cluster id")
	}
	defer rows.Close()

	entities, err := pgx.CollectRows(rows, pgx.RowToStructByName[entity])
	if err != nil {
		return nil, errors.WrapFail(err, "collect event rows")
	}

	return slices.Map(entities, entity.ToDomain), nil
}

func (r *repo) MaxSimilarityByClusters(
	ctx context.Context,
	clusterIDs []domain.ClusterID,
	embedding []float32,
) (map[domain.ClusterID]float64, error) {
	query := `
		SELECT cluster_id, MAX(1 - (embedding <=> $1)) AS similarity
		FROM events
		WHERE cluster_id = ANY($2::uuid[])
		GROUP BY cluster_id
	`

	ids := slices.Map(clusterIDs, func(id domain.ClusterID) string { return string(id) })

	rows, err := r.client.Query(ctx, query, pgvector.NewVector(embedding), ids)
	if err != nil {
		return nil, errors.WrapFail(err, "query max similarity by clusters")
	}
	defer rows.Close()

	entities, err := pgx.CollectRows(rows, pgx.RowToStructByName[clusterSimilarityEntity])
	if err != nil {
		return nil, errors.WrapFail(err, "collect max similarity rows")
	}

	if len(entities) == 0 {
		return nil, nil
	}

	result := make(map[domain.ClusterID]float64, len(entities))
	for _, e := range entities {
		result[domain.ClusterID(e.ClusterID.String())] = e.Similarity
	}

	return result, nil
}

func (r *repo) DeleteEventsByClusterID(ctx context.Context, clusterID domain.ClusterID) error {
	query := `DELETE FROM events WHERE cluster_id = $1`

	_, err := r.client.Exec(ctx, query, clusterID)
	if err != nil {
		return errors.WrapFail(err, "delete events by cluster id")
	}

	return nil
}

type clusterSimilarityEntity struct {
	ClusterID  uuid.UUID `db:"cluster_id"`
	Similarity float64   `db:"similarity"`
}

type entity struct {
	EventID    uuid.UUID       `db:"event_id"`
	UserID     uuid.UUID       `db:"user_id"`
	Source     string          `db:"source"`
	OccurredAt time.Time       `db:"occurred_at"`
	Context    json.RawMessage `db:"context"`
	ClusterID  uuid.UUID       `db:"cluster_id"`
	Similarity float64         `db:"similarity"`
}

func (e entity) ToDomain() domain.Event {
	return domain.Event{
		ID:         domain.EventID(e.EventID.String()),
		UserID:     domain.UserID(e.UserID.String()),
		Source:     domain.ParseEventSource(e.Source),
		Context:    domain.NormalizedEventContext(e.Context),
		OccurredAt: e.OccurredAt,
		ClusterID:  domain.ClusterID(e.ClusterID.String()),
		Similarity: e.Similarity,
	}
}
