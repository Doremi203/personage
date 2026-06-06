package grpc

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"net/http"
	"time"

	"github.com/Doremi203/personage/backend/tasker/internal/domain"
)

type adminGenerationSettingsUseCase interface {
	GetGenerationSettings(ctx context.Context) (domain.GenerationSettings, error)
	UpdateGenerationSettings(ctx context.Context, update domain.GenerationSettingsUpdate) (domain.GenerationSettings, error)
}

type adminGenerationSettingsItem struct {
	MinSimilarity             float64   `json:"minSimilarity"`
	ClosedSimilarityThreshold float64   `json:"closedSimilarityThreshold"`
	TopK                      int       `json:"topK"`
	MaxEventCount             int       `json:"maxEventCount"`
	InactivityMinutes         int       `json:"inactivityMinutes"`
	BatchSize                 int       `json:"batchSize"`
	UpdatedAt                 time.Time `json:"updatedAt"`
}

func generationSettingsToAdminItem(s domain.GenerationSettings) adminGenerationSettingsItem {
	return adminGenerationSettingsItem{
		MinSimilarity:             s.MinSimilarity,
		ClosedSimilarityThreshold: s.ClosedSimilarityThreshold,
		TopK:                      s.TopK,
		MaxEventCount:             s.MaxEventCount,
		InactivityMinutes:         int(s.InactivityTimeout.Minutes()),
		BatchSize:                 s.BatchSize,
		UpdatedAt:                 s.UpdatedAt,
	}
}

type adminUpdateGenerationSettingsRequest struct {
	MinSimilarity             *float64 `json:"minSimilarity,omitempty"`
	ClosedSimilarityThreshold *float64 `json:"closedSimilarityThreshold,omitempty"`
	TopK                      *int     `json:"topK,omitempty"`
	MaxEventCount             *int     `json:"maxEventCount,omitempty"`
	InactivityMinutes         *int     `json:"inactivityMinutes,omitempty"`
	BatchSize                 *int     `json:"batchSize,omitempty"`
}

func (req adminUpdateGenerationSettingsRequest) isEmpty() bool {
	return req.MinSimilarity == nil &&
		req.ClosedSimilarityThreshold == nil &&
		req.TopK == nil &&
		req.MaxEventCount == nil &&
		req.InactivityMinutes == nil &&
		req.BatchSize == nil
}

func (req adminUpdateGenerationSettingsRequest) toDomain() domain.GenerationSettingsUpdate {
	return domain.GenerationSettingsUpdate{
		MinSimilarity:             req.MinSimilarity,
		ClosedSimilarityThreshold: req.ClosedSimilarityThreshold,
		TopK:                      req.TopK,
		MaxEventCount:             req.MaxEventCount,
		InactivityMinutes:         req.InactivityMinutes,
		BatchSize:                 req.BatchSize,
	}
}

func writeGenerationSettingsError(w http.ResponseWriter, err error) {
	if stderrors.Is(err, domain.ErrInvalidGenerationSettings) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Error(w, "internal error", http.StatusInternalServerError)
}

func NewAdminGetGenerationSettingsHandler(uc adminGenerationSettingsUseCase, apiKey string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !checkAdminKey(w, r, apiKey) {
			return
		}

		settings, err := uc.GetGenerationSettings(r.Context())
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"settings": generationSettingsToAdminItem(settings)})
	}
}

func NewAdminUpdateGenerationSettingsHandler(uc adminGenerationSettingsUseCase, apiKey string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !checkAdminKey(w, r, apiKey) {
			return
		}

		var req adminUpdateGenerationSettingsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if req.isEmpty() {
			http.Error(w, "at least one setting is required", http.StatusBadRequest)
			return
		}

		settings, err := uc.UpdateGenerationSettings(r.Context(), req.toDomain())
		if err != nil {
			writeGenerationSettingsError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"settings": generationSettingsToAdminItem(settings)})
	}
}
