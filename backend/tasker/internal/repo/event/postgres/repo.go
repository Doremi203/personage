package eventpostgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/Doremi203/personage/backend/libs/go/postgres"
	"github.com/Doremi203/personage/backend/libs/go/slices"
	"github.com/Doremi203/personage/backend/tasker/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pgvector/pgvector-go"
)

func NewRepo(client postgres.Client) *repo {
	return &repo{
		client: client,
	}
}

type repo struct {
	client postgres.Client
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
			created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		)
		ON CONFLICT (event_id) DO UPDATE SET
			source = EXCLUDED.source,
		    user_id = EXCLUDED.user_id,
			occurred_at = EXCLUDED.occurred_at,
			context = EXCLUDED.context,
			embedding = EXCLUDED.embedding,
			cluster_id = EXCLUDED.cluster_id,
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
		time.Now(),
	)

	if err != nil {
		return fmt.Errorf("upsert event: %w", err)
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
			cluster_id
		FROM events
		WHERE cluster_id = $1
		ORDER BY occurred_at ASC
	`

	rows, err := r.client.Query(ctx, query, clusterID)
	if err != nil {
		return nil, fmt.Errorf("get events by cluster id: %w", err)
	}
	defer rows.Close()

	entities, err := pgx.CollectRows(rows, pgx.RowToStructByName[entity])
	if err != nil {
		return nil, errors.WrapFail(err, "collect event rows")
	}

	return slices.Map(entities, entity.ToDomain), nil
}

type entity struct {
	EventID    uuid.UUID       `db:"event_id"`
	UserID     uuid.UUID       `db:"user_id"`
	Source     string          `db:"source"`
	OccurredAt time.Time       `db:"occurred_at"`
	Context    json.RawMessage `db:"context"`
	ClusterID  uuid.UUID       `db:"cluster_id"`
}

func (e entity) ToDomain() domain.Event {
	return domain.Event{
		ID:         domain.EventID(e.EventID.String()),
		UserID:     domain.UserID(e.UserID.String()),
		Source:     domain.ParseEventSource(e.Source),
		Context:    domain.NormalizedEventContext(e.Context),
		OccurredAt: e.OccurredAt,
		ClusterID:  domain.ClusterID(e.ClusterID.String()),
	}
}
