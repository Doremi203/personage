package generationsettingspostgres

import (
	"context"
	"time"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/Doremi203/personage/backend/libs/go/postgres"
	"github.com/Doremi203/personage/backend/tasker/internal/domain"
	"github.com/jackc/pgx/v5"
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

func (r *repo) GetGenerationSettings(ctx context.Context) (domain.GenerationSettings, error) {
	query := `
		SELECT min_similarity, closed_similarity_threshold, top_k, max_event_count, inactivity_minutes, batch_size, task_duplicate_threshold, llm_model, updated_at
		FROM generation_settings
		WHERE id = 1
	`

	rows, err := r.client.Query(ctx, query)
	if err != nil {
		return domain.GenerationSettings{}, errors.WrapFail(err, "query generation settings")
	}
	defer rows.Close()

	entity, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[generationSettingsEntity])
	if err != nil {
		return domain.GenerationSettings{}, errors.WrapFail(err, "collect generation settings row")
	}

	return entity.ToDomain(), nil
}

func (r *repo) UpdateGenerationSettings(
	ctx context.Context,
	update domain.GenerationSettingsUpdate,
) (domain.GenerationSettings, error) {
	query := `
		UPDATE generation_settings
		SET min_similarity              = COALESCE($2, min_similarity),
		    closed_similarity_threshold = COALESCE($3, closed_similarity_threshold),
		    top_k                       = COALESCE($4, top_k),
		    max_event_count             = COALESCE($5, max_event_count),
		    inactivity_minutes          = COALESCE($6, inactivity_minutes),
		    batch_size                  = COALESCE($7, batch_size),
		    task_duplicate_threshold    = COALESCE($8, task_duplicate_threshold),
		    llm_model                   = COALESCE($9, llm_model),
		    updated_at                  = $1
		WHERE id = 1
		RETURNING min_similarity, closed_similarity_threshold, top_k, max_event_count, inactivity_minutes, batch_size, task_duplicate_threshold, llm_model, updated_at
	`

	rows, err := r.client.Query(
		ctx,
		query,
		r.clock(),
		update.MinSimilarity,
		update.ClosedSimilarityThreshold,
		update.TopK,
		update.MaxEventCount,
		update.InactivityMinutes,
		update.BatchSize,
		update.TaskDuplicateThreshold,
		update.LLMModel,
	)
	if err != nil {
		return domain.GenerationSettings{}, errors.WrapFail(err, "update generation settings")
	}
	defer rows.Close()

	entity, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[generationSettingsEntity])
	if err != nil {
		return domain.GenerationSettings{}, errors.WrapFail(err, "collect updated generation settings row")
	}

	return entity.ToDomain(), nil
}

type generationSettingsEntity struct {
	MinSimilarity             float64   `db:"min_similarity"`
	ClosedSimilarityThreshold float64   `db:"closed_similarity_threshold"`
	TopK                      int       `db:"top_k"`
	MaxEventCount             int       `db:"max_event_count"`
	InactivityMinutes         int       `db:"inactivity_minutes"`
	BatchSize                 int       `db:"batch_size"`
	TaskDuplicateThreshold    float64   `db:"task_duplicate_threshold"`
	LLMModel                  string    `db:"llm_model"`
	UpdatedAt                 time.Time `db:"updated_at"`
}

func (e generationSettingsEntity) ToDomain() domain.GenerationSettings {
	return domain.GenerationSettings{
		MinSimilarity:             e.MinSimilarity,
		ClosedSimilarityThreshold: e.ClosedSimilarityThreshold,
		TopK:                      e.TopK,
		MaxEventCount:             e.MaxEventCount,
		InactivityTimeout:         time.Duration(e.InactivityMinutes) * time.Minute,
		BatchSize:                 e.BatchSize,
		TaskDuplicateThreshold:    e.TaskDuplicateThreshold,
		LLMModel:                  e.LLMModel,
		UpdatedAt:                 e.UpdatedAt,
	}
}
