package grpc

import (
	stderrors "errors"
	"net/http"
	"time"

	"github.com/Doremi203/personage/backend/tasker/internal/domain"
)

type adminClusterListItem struct {
	ID                string    `json:"id"`
	UserID            string    `json:"userId"`
	Status            string    `json:"status"`
	EventCount        int       `json:"eventCount"`
	GenerationOutcome *string   `json:"generationOutcome,omitzero"`
	GenerationReason  *string   `json:"generationReason,omitzero"`
	TaskID            *string   `json:"taskId,omitzero"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

type adminClusterEventItem struct {
	ID         string    `json:"id"`
	UserID     string    `json:"userId"`
	ClusterID  string    `json:"clusterId"`
	Source     string    `json:"source"`
	OccurredAt time.Time `json:"occurredAt"`
	Context    string    `json:"context"`
	Similarity float64   `json:"similarity"`
}

func clusterToAdminItem(c domain.AdminClusterListItem) adminClusterListItem {
	var taskID *string
	if c.TaskID != nil {
		s := c.TaskID.String()
		taskID = &s
	}
	return adminClusterListItem{
		ID:                c.ClusterID.String(),
		UserID:            c.UserID.String(),
		Status:            string(c.Status),
		EventCount:        c.EventCount,
		GenerationOutcome: generationOutcomeStr(c.GenerationOutcome),
		GenerationReason:  c.GenerationReason,
		TaskID:            taskID,
		CreatedAt:         c.CreatedAt,
		UpdatedAt:         c.UpdatedAt,
	}
}

func eventToAdminItem(e domain.Event) adminClusterEventItem {
	return adminClusterEventItem{
		ID:         e.ID.String(),
		UserID:     e.UserID.String(),
		ClusterID:  e.ClusterID.String(),
		Source:     e.Source.String(),
		OccurredAt: e.OccurredAt,
		Context:    string(e.Context),
		Similarity: e.Similarity,
	}
}

func NewAdminListClustersHandler(uc adminClustersUseCase, apiKey string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !checkAdminKey(w, r, apiKey) {
			return
		}

		userID := r.PathValue("userId")
		if userID == "" {
			http.Error(w, "userId is required", http.StatusBadRequest)
			return
		}

		clusters, err := uc.ListClustersForUser(r.Context(), domain.UserID(userID))
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		items := make([]adminClusterListItem, 0, len(clusters))
		for _, c := range clusters {
			items = append(items, clusterToAdminItem(c))
		}

		writeJSON(w, http.StatusOK, map[string]any{"clusters": items})
	}
}

func NewAdminGetClusterHandler(uc adminClustersUseCase, apiKey string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !checkAdminKey(w, r, apiKey) {
			return
		}

		clusterID := r.PathValue("clusterId")
		if clusterID == "" {
			http.Error(w, "clusterId is required", http.StatusBadRequest)
			return
		}

		cluster, err := uc.GetCluster(r.Context(), domain.ClusterID(clusterID))
		if err != nil {
			if stderrors.Is(err, domain.ErrClusterNotFound) {
				http.Error(w, "cluster not found", http.StatusNotFound)
				return
			}
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"cluster": clusterToAdminItem(cluster)})
	}
}

func NewAdminListClusterEventsHandler(uc adminClustersUseCase, apiKey string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !checkAdminKey(w, r, apiKey) {
			return
		}

		clusterID := r.PathValue("clusterId")
		if clusterID == "" {
			http.Error(w, "clusterId is required", http.StatusBadRequest)
			return
		}

		events, err := uc.ListClusterEvents(r.Context(), domain.ClusterID(clusterID))
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		items := make([]adminClusterEventItem, 0, len(events))
		for _, e := range events {
			items = append(items, eventToAdminItem(e))
		}

		writeJSON(w, http.StatusOK, map[string]any{"events": items})
	}
}
