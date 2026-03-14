package clusterpostgres

import (
	"context"
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

func (r *repo) FindSimilarClusters(
	ctx context.Context,
	userID domain.UserID,
	embedding []float32,
	topK int,
) ([]domain.ClusterWithSimilarity, error) {
	query := `
		SELECT 
			cluster_id,
			user_id,
			centroid,
			event_count,
			status,
			created_at,
			updated_at,
			1 - (centroid <=> $1) as similarity
		FROM clusters
		WHERE user_id = $2 AND status = $3
		ORDER BY centroid <=> $1
		LIMIT $4
	`

	rows, err := r.client.Query(ctx, query,
		pgvector.NewVector(embedding),
		userID,
		domain.ClusterStatusOpen,
		topK,
	)
	if err != nil {
		return nil, errors.WrapFail(err, "find similar clusters")
	}
	defer rows.Close()

	entities, err := pgx.CollectRows(rows, pgx.RowToStructByName[clusterEntityWithSimilarity])
	if err != nil {
		return nil, errors.WrapFail(err, "collect cluster rows")
	}

	return slices.Map(entities, clusterEntityWithSimilarity.ToDomain), nil
}

func (r *repo) UpsertCluster(ctx context.Context, cluster domain.Cluster) error {
	query := `
		INSERT INTO clusters (
			cluster_id,
			user_id,
			centroid,
			event_count,
			status,
			created_at,
			updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7
		)
		ON CONFLICT (cluster_id) DO UPDATE SET
			centroid = EXCLUDED.centroid,
			event_count = EXCLUDED.event_count,
			status = EXCLUDED.status,
			updated_at = EXCLUDED.updated_at
	`

	_, err := r.client.Exec(ctx, query,
		cluster.ID,
		cluster.UserID,
		pgvector.NewVector(cluster.Centroid),
		cluster.EventCount,
		cluster.Status,
		cluster.CreatedAt,
		cluster.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("upsert cluster: %w", err)
	}

	return nil
}

func (r *repo) FindClosableClusters(
	ctx context.Context,
	maxEventCount int,
	inactivityDuration time.Duration,
	limit int,
) ([]domain.Cluster, error) {
	query := `
		SELECT 
			cluster_id,
			user_id,
			centroid,
			event_count,
			status,
			created_at,
			updated_at
		FROM clusters
		WHERE status = $1
		AND (event_count >= $2 OR updated_at < $3)
		ORDER BY updated_at ASC
		LIMIT $4
		FOR UPDATE SKIP LOCKED
	`

	inactivityThreshold := time.Now().Add(-inactivityDuration)

	rows, err := r.client.Query(ctx, query,
		domain.ClusterStatusOpen,
		maxEventCount,
		inactivityThreshold,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("find closable clusters: %w", err)
	}
	defer rows.Close()

	entities, err := pgx.CollectRows(rows, pgx.RowToStructByName[clusterEntity])
	if err != nil {
		return nil, errors.WrapFail(err, "collect cluster rows")
	}

	return slices.Map(entities, clusterEntity.ToDomain), nil
}

func (r *repo) UpdateClusterStatus(ctx context.Context, clusterID domain.ClusterID, status domain.ClusterStatus) error {
	query := `
		UPDATE clusters
		SET status = $1, updated_at = $2
		WHERE cluster_id = $3
	`

	_, err := r.client.Exec(ctx, query, status, time.Now(), clusterID)
	if err != nil {
		return fmt.Errorf("update cluster status: %w", err)
	}

	return nil
}

func (r *repo) DeleteCluster(ctx context.Context, clusterID domain.ClusterID) error {
	query := `DELETE FROM clusters WHERE cluster_id = $1`

	result, err := r.client.Exec(ctx, query, clusterID)
	if err != nil {
		return errors.WrapFail(err, "delete cluster")
	}

	if result.RowsAffected() == 0 {
		return errors.Errorf("cluster not found: %s", clusterID)
	}

	return nil
}

func (r *repo) RecoverStaleClusters(ctx context.Context, staleThreshold time.Duration) (int, error) {
	query := `
		UPDATE clusters
		SET status = $1, updated_at = $2
		WHERE status = $3
		AND updated_at < $4
	`

	now := time.Now()
	threshold := now.Add(-staleThreshold)

	result, err := r.client.Exec(ctx, query,
		domain.ClusterStatusOpen,
		now,
		domain.ClusterStatusProcessing,
		threshold,
	)
	if err != nil {
		return 0, fmt.Errorf("recover stale clusters: %w", err)
	}

	return int(result.RowsAffected()), nil
}

type clusterEntity struct {
	ClusterID  uuid.UUID       `db:"cluster_id"`
	UserID     uuid.UUID       `db:"user_id"`
	Centroid   pgvector.Vector `db:"centroid"`
	EventCount int             `db:"event_count"`
	Status     string          `db:"status"`
	CreatedAt  time.Time       `db:"created_at"`
	UpdatedAt  time.Time       `db:"updated_at"`
}

type clusterEntityWithSimilarity struct {
	clusterEntity
	Similarity float64 `db:"similarity"`
}

func (e clusterEntity) ToDomain() domain.Cluster {
	return domain.Cluster{
		ID:         domain.ClusterID(e.ClusterID.String()),
		UserID:     domain.UserID(e.UserID.String()),
		Centroid:   e.Centroid.Slice(),
		EventCount: e.EventCount,
		Status:     domain.ClusterStatus(e.Status),
		CreatedAt:  e.CreatedAt,
		UpdatedAt:  e.UpdatedAt,
	}
}

func (e clusterEntityWithSimilarity) ToDomain() domain.ClusterWithSimilarity {
	return domain.ClusterWithSimilarity{
		Cluster:    e.clusterEntity.ToDomain(),
		Similarity: e.Similarity,
	}
}
