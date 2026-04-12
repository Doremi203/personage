package grpc

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/Doremi203/personage/backend/tasker/internal/domain"
	"github.com/google/uuid"
)

type testTaskCreator interface {
	CreateTask(ctx context.Context, task domain.Task) error
}

type testCreateTaskRequest struct {
	UserID      string `json:"user_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`   // "unplanned" | "planned" | "completed"; defaults to "unplanned"
	Priority    int    `json:"priority"` // 1–10; defaults to 5
	Category    string `json:"category"` // "work" | "study" | "personal"; defaults to "personal"
}

type testCreateTaskResponse struct {
	ID string `json:"id"`
}

// NewTestCreateTaskHandler returns an HTTP handler for POST /v1/test/tasks.
// It inserts a task directly via the repo, bypassing the AI pipeline.
// cluster_id is left nil (allowed after migration 00004).
// Must only be registered in non-production environments.
func NewTestCreateTaskHandler(repo testTaskCreator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req testCreateTaskRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if req.UserID == "" || req.Title == "" {
			http.Error(w, "user_id and title are required", http.StatusBadRequest)
			return
		}

		validStatuses := map[string]domain.TaskStatus{
			"unplanned": domain.TaskStatusUnplanned,
			"planned":   domain.TaskStatusPlanned,
			"completed": domain.TaskStatusCompleted,
		}
		status := domain.TaskStatusUnplanned
		if req.Status != "" {
			s, ok := validStatuses[req.Status]
			if !ok {
				http.Error(w, "invalid status: must be unplanned, planned, or completed", http.StatusBadRequest)
				return
			}
			status = s
		}

		priority := 5
		if req.Priority != 0 {
			if req.Priority < 1 || req.Priority > 10 {
				http.Error(w, "invalid priority: must be between 1 and 10", http.StatusBadRequest)
				return
			}
			priority = req.Priority
		}

		category := domain.TaskCategoryPersonal
		if req.Category != "" {
			category = domain.NewTaskCategory(req.Category)
			if req.Category != string(category) {
				http.Error(w, "invalid category: must be work, study, or personal", http.StatusBadRequest)
				return
			}
		}

		now := time.Now()
		task := domain.Task{
			ID:          domain.TaskID(uuid.New().String()),
			UserID:      domain.UserID(req.UserID),
			ClusterID:   nil,
			Title:       req.Title,
			Description: req.Description,
			Status:      status,
			Priority:    priority,
			Category:    category,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		if err := repo.CreateTask(r.Context(), task); err != nil {
			http.Error(w, "failed to create task", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(testCreateTaskResponse{ID: task.ID.String()})
	}
}
