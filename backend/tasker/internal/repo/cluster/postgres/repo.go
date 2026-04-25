package clusterpostgres

import (
	"context"
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
			generation_outcome,
			generation_reason,
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
		return errors.WrapFail(err, "upsert cluster")
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
			generation_outcome,
			generation_reason,
			created_at,
			updated_at
		FROM clusters
		WHERE status = $1
		AND (event_count >= $2 OR updated_at < $3)
		ORDER BY updated_at ASC
		LIMIT $4
		FOR UPDATE SKIP LOCKED
	`

	inactivityThreshold := r.clock().Add(-inactivityDuration)

	rows, err := r.client.Query(ctx, query,
		domain.ClusterStatusOpen,
		maxEventCount,
		inactivityThreshold,
		limit,
	)
	if err != nil {
		return nil, errors.WrapFail(err, "find closable clusters")
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

	_, err := r.client.Exec(ctx, query, status, r.clock(), clusterID)
	if err != nil {
		return errors.WrapFail(err, "update cluster status")
	}

	return nil
}

func (r *repo) FinalizeCluster(
	ctx context.Context,
	clusterID domain.ClusterID,
	outcome domain.ClusterGenerationOutcome,
	reason *string,
) error {
	query := `
		UPDATE clusters
		SET status = $1,
			generation_outcome = $2,
			generation_reason = $3,
			updated_at = $4
		WHERE cluster_id = $5
	`

	_, err := r.client.Exec(ctx, query, domain.ClusterStatusClosed, outcome, reason, r.clock(), clusterID)
	if err != nil {
		return errors.WrapFail(err, "finalize cluster")
	}

	return nil
}

func (r *repo) ListGenerationDiagnosticsByUserID(
	ctx context.Context,
	userID domain.UserID,
) ([]domain.ClusterGenerationDiagnostic, error) {
	query := `
		SELECT
			c.cluster_id,
			c.user_id,
			c.status,
			c.event_count,
			c.generation_outcome,
			c.generation_reason,
			COUNT(t.task_id) AS generated_task_count,
			c.created_at,
			c.updated_at
		FROM clusters c
		LEFT JOIN tasks t ON t.cluster_id = c.cluster_id
		WHERE c.user_id = $1
		GROUP BY
			c.cluster_id,
			c.user_id,
			c.status,
			c.event_count,
			c.generation_outcome,
			c.generation_reason,
			c.created_at,
			c.updated_at
		ORDER BY c.created_at ASC
	`

	rows, err := r.client.Query(ctx, query, userID)
	if err != nil {
		return nil, errors.WrapFail(err, "list cluster generation diagnostics")
	}
	defer rows.Close()

	entities, err := pgx.CollectRows(rows, pgx.RowToStructByName[clusterGenerationDiagnosticEntity])
	if err != nil {
		return nil, errors.WrapFail(err, "collect cluster generation diagnostics")
	}

	return slices.Map(entities, clusterGenerationDiagnosticEntity.ToDomain), nil
}

func (r *repo) DeleteCluster(ctx context.Context, clusterID domain.ClusterID) error {
	query := `DELETE FROM clusters WHERE cluster_id = $1`

	result, err := r.client.Exec(ctx, query, clusterID)
	if err != nil {
		return errors.WrapFail(err, "delete cluster")
	}

	if result.RowsAffected() == 0 {
		return errors.Errorf("cluster not found %v", errors.Token("cluster_id", clusterID))
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

	now := r.clock()
	threshold := now.Add(-staleThreshold)

	result, err := r.client.Exec(ctx, query,
		domain.ClusterStatusOpen,
		now,
		domain.ClusterStatusProcessing,
		threshold,
	)
	if err != nil {
		return 0, errors.WrapFail(err, "recover stale clusters")
	}

	return int(result.RowsAffected()), nil
}

type clusterEntity struct {
	ClusterID         uuid.UUID       `db:"cluster_id"`
	UserID            uuid.UUID       `db:"user_id"`
	Centroid          pgvector.Vector `db:"centroid"`
	EventCount        int             `db:"event_count"`
	Status            string          `db:"status"`
	GenerationOutcome *string         `db:"generation_outcome"`
	GenerationReason  *string         `db:"generation_reason"`
	CreatedAt         time.Time       `db:"created_at"`
	UpdatedAt         time.Time       `db:"updated_at"`
}

type clusterEntityWithSimilarity struct {
	clusterEntity
	Similarity float64 `db:"similarity"`
}

func (e clusterEntity) ToDomain() domain.Cluster {
	var outcome *domain.ClusterGenerationOutcome
	if e.GenerationOutcome != nil {
		value := domain.ClusterGenerationOutcome(*e.GenerationOutcome)
		outcome = &value
	}

	return domain.Cluster{
		ID:                domain.ClusterID(e.ClusterID.String()),
		UserID:            domain.UserID(e.UserID.String()),
		Centroid:          e.Centroid.Slice(),
		EventCount:        e.EventCount,
		Status:            domain.ClusterStatus(e.Status),
		GenerationOutcome: outcome,
		GenerationReason:  e.GenerationReason,
		CreatedAt:         e.CreatedAt,
		UpdatedAt:         e.UpdatedAt,
	}
}

func (e clusterEntityWithSimilarity) ToDomain() domain.ClusterWithSimilarity {
	return domain.ClusterWithSimilarity{
		Cluster:    e.clusterEntity.ToDomain(),
		Similarity: e.Similarity,
	}
}

type clusterGenerationDiagnosticEntity struct {
	ClusterID          uuid.UUID `db:"cluster_id"`
	UserID             uuid.UUID `db:"user_id"`
	Status             string    `db:"status"`
	EventCount         int       `db:"event_count"`
	GenerationOutcome  *string   `db:"generation_outcome"`
	GenerationReason   *string   `db:"generation_reason"`
	GeneratedTaskCount int       `db:"generated_task_count"`
	CreatedAt          time.Time `db:"created_at"`
	UpdatedAt          time.Time `db:"updated_at"`
}

func (e clusterGenerationDiagnosticEntity) ToDomain() domain.ClusterGenerationDiagnostic {
	var outcome *domain.ClusterGenerationOutcome
	if e.GenerationOutcome != nil {
		value := domain.ClusterGenerationOutcome(*e.GenerationOutcome)
		outcome = &value
	}

	return domain.ClusterGenerationDiagnostic{
		ClusterID:          domain.ClusterID(e.ClusterID.String()),
		UserID:             domain.UserID(e.UserID.String()),
		Status:             domain.ClusterStatus(e.Status),
		EventCount:         e.EventCount,
		GenerationOutcome:  outcome,
		GenerationReason:   e.GenerationReason,
		GeneratedTaskCount: e.GeneratedTaskCount,
		CreatedAt:          e.CreatedAt,
		UpdatedAt:          e.UpdatedAt,
	}
}
