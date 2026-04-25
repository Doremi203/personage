package grpc

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/Doremi203/personage/backend/tasker/internal/domain"
	"github.com/google/uuid"
)

type testTaskLister interface {
	GetTasksByUserID(ctx context.Context, userID domain.UserID) ([]domain.Task, error)
}

type testListTaskItem struct {
	ID               string     `json:"id"`
	UserID           string     `json:"user_id"`
	ClusterID        *string    `json:"cluster_id,omitzero"`
	Title            string     `json:"title"`
	Description      string     `json:"description"`
	DurationMinutes  int        `json:"duration_minutes"`
	Priority         int        `json:"priority"`
	Deadline         *time.Time `json:"deadline,omitzero"`
	StartTime        *time.Time `json:"start_time,omitzero"`
	EndTime          *time.Time `json:"end_time,omitzero"`
	Status           string     `json:"status"`
	Category         string     `json:"category"`
	EvidenceEventIDs []string   `json:"evidence_event_ids,omitzero"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type testClusterGenerationDiagnosticsLister interface {
	ListGenerationDiagnosticsByUserID(ctx context.Context, userID domain.UserID) ([]domain.ClusterGenerationDiagnostic, error)
}

type testClusterGenerationDiagnosticItem struct {
	ClusterID          string    `json:"cluster_id"`
	UserID             string    `json:"user_id"`
	Status             string    `json:"status"`
	EventCount         int       `json:"event_count"`
	GenerationOutcome  *string   `json:"generation_outcome,omitzero"`
	GenerationReason   *string   `json:"generation_reason,omitzero"`
	GeneratedTaskCount int       `json:"generated_task_count"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

func clusterIDStr(c *domain.ClusterID) *string {
	if c == nil {
		return nil
	}
	s := c.String()
	return &s
}

func generationOutcomeStr(outcome *domain.ClusterGenerationOutcome) *string {
	if outcome == nil {
		return nil
	}

	s := string(*outcome)
	return &s
}

func evidenceEventIDs(ids []domain.EventID) []string {
	values := make([]string, len(ids))
	for i, id := range ids {
		values[i] = id.String()
	}

	return values
}

func NewTestListTasksHandler(repo testTaskLister) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		userIDStr := r.URL.Query().Get("user_id")
		if userIDStr == "" {
			http.Error(w, "user_id is required", http.StatusBadRequest)
			return
		}

		tasks, err := repo.GetTasksByUserID(r.Context(), domain.UserID(userIDStr))
		if err != nil {
			http.Error(w, "failed to list tasks", http.StatusInternalServerError)
			return
		}

		items := make([]testListTaskItem, 0, len(tasks))
		for _, t := range tasks {
			items = append(items, testListTaskItem{
				ID:               t.ID.String(),
				UserID:           t.UserID.String(),
				ClusterID:        clusterIDStr(t.ClusterID),
				Title:            t.Title,
				Description:      t.Description,
				DurationMinutes:  int(t.Duration.Minutes()),
				Priority:         t.Priority,
				Deadline:         t.Deadline,
				StartTime:        t.StartTime,
				EndTime:          t.EndTime,
				Status:           string(t.Status),
				Category:         string(t.Category),
				EvidenceEventIDs: evidenceEventIDs(t.EvidenceEventIDs),
				CreatedAt:        t.CreatedAt,
				UpdatedAt:        t.UpdatedAt,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(items)
	}
}

func NewTestListClusterGenerationDiagnosticsHandler(repo testClusterGenerationDiagnosticsLister) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		userIDStr := r.URL.Query().Get("user_id")
		if userIDStr == "" {
			http.Error(w, "user_id is required", http.StatusBadRequest)
			return
		}

		diagnostics, err := repo.ListGenerationDiagnosticsByUserID(r.Context(), domain.UserID(userIDStr))
		if err != nil {
			http.Error(w, "failed to list cluster diagnostics", http.StatusInternalServerError)
			return
		}

		items := make([]testClusterGenerationDiagnosticItem, 0, len(diagnostics))
		for _, diagnostic := range diagnostics {
			items = append(items, testClusterGenerationDiagnosticItem{
				ClusterID:          diagnostic.ClusterID.String(),
				UserID:             diagnostic.UserID.String(),
				Status:             string(diagnostic.Status),
				EventCount:         diagnostic.EventCount,
				GenerationOutcome:  generationOutcomeStr(diagnostic.GenerationOutcome),
				GenerationReason:   diagnostic.GenerationReason,
				GeneratedTaskCount: diagnostic.GeneratedTaskCount,
				CreatedAt:          diagnostic.CreatedAt,
				UpdatedAt:          diagnostic.UpdatedAt,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(items)
	}
}

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

func NewTestCreateTaskHandler(repo testTaskCreator, clock func() time.Time) http.HandlerFunc {
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

		now := clock()
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
