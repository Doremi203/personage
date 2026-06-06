package domain

import (
	"context"
	"time"

	"github.com/Doremi203/personage/backend/libs/go/errors"
)

var ErrInvalidGenerationSettings = errors.Error("invalid generation settings")

type GenerationSettings struct {
	MinSimilarity             float64
	ClosedSimilarityThreshold float64
	TopK                      int
	MaxEventCount             int
	InactivityTimeout         time.Duration
	BatchSize                 int
	UpdatedAt                 time.Time
}

type GenerationSettingsUpdate struct {
	MinSimilarity             *float64
	ClosedSimilarityThreshold *float64
	TopK                      *int
	MaxEventCount             *int
	InactivityMinutes         *int
	BatchSize                 *int
}

func (u GenerationSettingsUpdate) Validate() error {
	if u.MinSimilarity != nil && (*u.MinSimilarity <= 0 || *u.MinSimilarity > 1) {
		return errors.WrapFailf(
			ErrInvalidGenerationSettings,
			"min_similarity must be in (0,1] %s",
			errors.Token("min_similarity", *u.MinSimilarity),
		)
	}
	if u.ClosedSimilarityThreshold != nil && (*u.ClosedSimilarityThreshold <= 0 || *u.ClosedSimilarityThreshold > 1) {
		return errors.WrapFailf(
			ErrInvalidGenerationSettings,
			"closed_similarity_threshold must be in (0,1] %s",
			errors.Token("closed_similarity_threshold", *u.ClosedSimilarityThreshold),
		)
	}
	if u.TopK != nil && *u.TopK < 1 {
		return errors.WrapFailf(
			ErrInvalidGenerationSettings,
			"top_k must be >= 1 %s",
			errors.Token("top_k", *u.TopK),
		)
	}
	if u.MaxEventCount != nil && *u.MaxEventCount < 1 {
		return errors.WrapFailf(
			ErrInvalidGenerationSettings,
			"max_event_count must be >= 1 %s",
			errors.Token("max_event_count", *u.MaxEventCount),
		)
	}
	if u.InactivityMinutes != nil && *u.InactivityMinutes < 1 {
		return errors.WrapFailf(
			ErrInvalidGenerationSettings,
			"inactivity_minutes must be >= 1 %s",
			errors.Token("inactivity_minutes", *u.InactivityMinutes),
		)
	}
	if u.BatchSize != nil && *u.BatchSize < 1 {
		return errors.WrapFailf(
			ErrInvalidGenerationSettings,
			"batch_size must be >= 1 %s",
			errors.Token("batch_size", *u.BatchSize),
		)
	}
	return nil
}

//go:generate mockgen -source=generation_settings.go -destination=mock/generation_settings_mock.go -typed

type GenerationSettingsRepo interface {
	GetGenerationSettings(ctx context.Context) (GenerationSettings, error)
	UpdateGenerationSettings(ctx context.Context, update GenerationSettingsUpdate) (GenerationSettings, error)
}

type GenerationSettingsProvider interface {
	GenerationSettings(ctx context.Context) (GenerationSettings, error)
}
