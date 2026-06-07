package grpc

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	domerrors "github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/Doremi203/personage/backend/tasker/internal/domain"
	"github.com/google/uuid"
)

type adminTasksUseCase interface {
	ListTasks(ctx context.Context, userID domain.UserID) ([]domain.Task, error)
	GetTask(ctx context.Context, taskID domain.TaskID, userID domain.UserID) (domain.Task, error)
	CreateTask(ctx context.Context, task domain.Task) (domain.Task, error)
	UpdateTask(ctx context.Context, taskID domain.TaskID, userID domain.UserID, update domain.TaskUpdate) (domain.Task, error)
	Approve(ctx context.Context, taskID domain.TaskID, userID domain.UserID) (domain.Task, error)
	ListModeratedUsers(ctx context.Context) ([]domain.UserID, error)
	SetUserModeration(ctx context.Context, userID domain.UserID, enabled bool) error
}

type adminClustersUseCase interface {
	ListClustersForUser(ctx context.Context, userID domain.UserID) ([]domain.AdminClusterListItem, error)
	GetCluster(ctx context.Context, clusterID domain.ClusterID) (domain.AdminClusterListItem, error)
	ListClusterEvents(ctx context.Context, clusterID domain.ClusterID) ([]domain.Event, error)
}

type adminPromptsUseCase interface {
	ListPrompts(ctx context.Context) ([]domain.Prompt, error)
	GetPrompt(ctx context.Context, id domain.PromptID) (domain.Prompt, error)
	UpdatePrompt(ctx context.Context, id domain.PromptID, update domain.PromptUpdate) (domain.Prompt, error)
}

type adminTaskItem struct {
	ID              string     `json:"id"`
	UserID          string     `json:"userId"`
	ClusterID       *string    `json:"clusterId,omitzero"`
	Title           string     `json:"title"`
	Description     string     `json:"description"`
	DurationMinutes int        `json:"durationMinutes"`
	Priority        int        `json:"priority"`
	Deadline        *time.Time `json:"deadline,omitzero"`
	StartTime       *time.Time `json:"startTime,omitzero"`
	EndTime         *time.Time `json:"endTime,omitzero"`
	Status          string     `json:"status"`
	Category        string     `json:"category"`
	IsApproved      bool       `json:"isApproved"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

func taskToAdminItem(t domain.Task) adminTaskItem {
	return adminTaskItem{
		ID:              t.ID.String(),
		UserID:          t.UserID.String(),
		ClusterID:       clusterIDStr(t.ClusterID),
		Title:           t.Title,
		Description:     t.Description,
		DurationMinutes: int(t.Duration.Minutes()),
		Priority:        t.Priority,
		Deadline:        t.Deadline,
		StartTime:       t.StartTime,
		EndTime:         t.EndTime,
		Status:          string(t.Status),
		Category:        string(t.Category),
		IsApproved:      t.IsApproved,
		CreatedAt:       t.CreatedAt,
		UpdatedAt:       t.UpdatedAt,
	}
}

type adminUpdateTaskRequest struct {
	Title       *string    `json:"title,omitempty"`
	Description *string    `json:"description,omitempty"`
	StartTime   *time.Time `json:"startTime,omitempty"`
	EndTime     *time.Time `json:"endTime,omitempty"`
	Deadline    *time.Time `json:"deadline,omitempty"`
	Priority    *int       `json:"priority,omitempty"`
	Status      *string    `json:"status,omitempty"`
	Category    *string    `json:"category,omitempty"`
	IsApproved  *bool      `json:"isApproved,omitempty"`
}

func (req adminUpdateTaskRequest) toDomain() (domain.TaskUpdate, error) {
	update := domain.TaskUpdate{
		Title:       req.Title,
		Description: req.Description,
		StartTime:   req.StartTime,
		EndTime:     req.EndTime,
		Deadline:    req.Deadline,
		Priority:    req.Priority,
		IsApproved:  req.IsApproved,
	}

	if req.Status != nil {
		switch s := domain.TaskStatus(*req.Status); s {
		case domain.TaskStatusUnplanned, domain.TaskStatusPlanned, domain.TaskStatusCompleted:
			update.Status = &s
		default:
			return domain.TaskUpdate{}, domerrors.Errorf("invalid status %v", domerrors.Token("value", *req.Status))
		}
	}

	if req.Category != nil {
		switch c := domain.TaskCategory(*req.Category); c {
		case domain.TaskCategoryWork, domain.TaskCategoryStudy, domain.TaskCategoryPersonal:
			update.Category = &c
		default:
			return domain.TaskUpdate{}, domerrors.Errorf("invalid category %v", domerrors.Token("value", *req.Category))
		}
	}

	if req.Priority != nil && (*req.Priority < 1 || *req.Priority > 10) {
		return domain.TaskUpdate{}, domerrors.Errorf("invalid priority %v: must be 1-10", domerrors.Token("value", *req.Priority))
	}

	return update, nil
}

type adminCreateTaskRequest struct {
	Title           string     `json:"title"`
	Description     string     `json:"description"`
	DurationMinutes int        `json:"durationMinutes"`
	Priority        int        `json:"priority"`
	Deadline        *time.Time `json:"deadline,omitempty"`
	StartTime       *time.Time `json:"startTime,omitempty"`
	EndTime         *time.Time `json:"endTime,omitempty"`
	Status          string     `json:"status"`
	Category        string     `json:"category"`
	IsApproved      *bool      `json:"isApproved,omitempty"`
}

func (req adminCreateTaskRequest) toDomain(id domain.TaskID, userID domain.UserID, now time.Time) (domain.Task, error) {
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return domain.Task{}, domerrors.Error("title is required")
	}

	status := domain.TaskStatusUnplanned
	if req.Status != "" {
		switch s := domain.TaskStatus(req.Status); s {
		case domain.TaskStatusUnplanned, domain.TaskStatusPlanned, domain.TaskStatusCompleted:
			status = s
		default:
			return domain.Task{}, domerrors.Errorf("invalid status %v", domerrors.Token("value", req.Status))
		}
	}

	category := domain.TaskCategoryPersonal
	if req.Category != "" {
		switch c := domain.TaskCategory(req.Category); c {
		case domain.TaskCategoryWork, domain.TaskCategoryStudy, domain.TaskCategoryPersonal:
			category = c
		default:
			return domain.Task{}, domerrors.Errorf("invalid category %v", domerrors.Token("value", req.Category))
		}
	}

	priority := 5
	if req.Priority != 0 {
		if req.Priority < 1 || req.Priority > 10 {
			return domain.Task{}, domerrors.Errorf("invalid priority %v: must be 1-10", domerrors.Token("value", req.Priority))
		}
		priority = req.Priority
	}

	if req.DurationMinutes < 0 {
		return domain.Task{}, domerrors.Errorf("invalid durationMinutes %v: must be >= 0", domerrors.Token("value", req.DurationMinutes))
	}

	isApproved := true
	if req.IsApproved != nil {
		isApproved = *req.IsApproved
	}

	return domain.Task{
		ID:          id,
		UserID:      userID,
		Title:       title,
		Description: req.Description,
		Duration:    time.Duration(req.DurationMinutes) * time.Minute,
		Priority:    priority,
		Deadline:    req.Deadline,
		StartTime:   req.StartTime,
		EndTime:     req.EndTime,
		Status:      status,
		Category:    category,
		IsApproved:  isApproved,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

type adminSetModerationRequest struct {
	Enabled bool `json:"enabled"`
}

type adminListModerationResponse struct {
	UserIDs []string `json:"userIds"`
}

func checkAdminKey(w http.ResponseWriter, r *http.Request, apiKey string) bool {
	if apiKey == "" || r.Header.Get("X-Admin-Key") != apiKey {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeTaskRepoError(w http.ResponseWriter, err error) {
	if errors.Is(err, domain.ErrTaskNotFound) {
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}
	http.Error(w, "internal error", http.StatusInternalServerError)
}

func NewAdminListTasksHandler(uc adminTasksUseCase, apiKey string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !checkAdminKey(w, r, apiKey) {
			return
		}

		userID := r.PathValue("userId")
		if userID == "" {
			http.Error(w, "userId is required", http.StatusBadRequest)
			return
		}

		tasks, err := uc.ListTasks(r.Context(), domain.UserID(userID))
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		items := make([]adminTaskItem, 0, len(tasks))
		for _, t := range tasks {
			items = append(items, taskToAdminItem(t))
		}

		writeJSON(w, http.StatusOK, map[string]any{"tasks": items})
	}
}

func NewAdminGetTaskHandler(uc adminTasksUseCase, apiKey string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !checkAdminKey(w, r, apiKey) {
			return
		}

		userID := r.PathValue("userId")
		taskID := r.PathValue("taskId")
		if userID == "" || taskID == "" {
			http.Error(w, "userId and taskId are required", http.StatusBadRequest)
			return
		}

		task, err := uc.GetTask(r.Context(), domain.TaskID(taskID), domain.UserID(userID))
		if err != nil {
			writeTaskRepoError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"task": taskToAdminItem(task)})
	}
}

func NewAdminCreateTaskHandler(uc adminTasksUseCase, apiKey string, clock func() time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !checkAdminKey(w, r, apiKey) {
			return
		}

		userID := r.PathValue("userId")
		if userID == "" {
			http.Error(w, "userId is required", http.StatusBadRequest)
			return
		}

		var req adminCreateTaskRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		task, err := req.toDomain(domain.TaskID(uuid.New().String()), domain.UserID(userID), clock())
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		created, err := uc.CreateTask(r.Context(), task)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusCreated, map[string]any{"task": taskToAdminItem(created)})
	}
}

func NewAdminUpdateTaskHandler(uc adminTasksUseCase, apiKey string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !checkAdminKey(w, r, apiKey) {
			return
		}

		userID := r.PathValue("userId")
		taskID := r.PathValue("taskId")
		if userID == "" || taskID == "" {
			http.Error(w, "userId and taskId are required", http.StatusBadRequest)
			return
		}

		var req adminUpdateTaskRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		update, err := req.toDomain()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		task, err := uc.UpdateTask(r.Context(), domain.TaskID(taskID), domain.UserID(userID), update)
		if err != nil {
			writeTaskRepoError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"task": taskToAdminItem(task)})
	}
}

func NewAdminApproveTaskHandler(uc adminTasksUseCase, apiKey string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !checkAdminKey(w, r, apiKey) {
			return
		}

		userID := r.PathValue("userId")
		taskID := r.PathValue("taskId")
		if userID == "" || taskID == "" {
			http.Error(w, "userId and taskId are required", http.StatusBadRequest)
			return
		}

		task, err := uc.Approve(r.Context(), domain.TaskID(taskID), domain.UserID(userID))
		if err != nil {
			writeTaskRepoError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"task": taskToAdminItem(task)})
	}
}

func NewAdminListModerationHandler(uc adminTasksUseCase, apiKey string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !checkAdminKey(w, r, apiKey) {
			return
		}

		userIDs, err := uc.ListModeratedUsers(r.Context())
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		ids := make([]string, 0, len(userIDs))
		for _, id := range userIDs {
			ids = append(ids, id.String())
		}

		writeJSON(w, http.StatusOK, adminListModerationResponse{UserIDs: ids})
	}
}

func NewAdminSetModerationHandler(uc adminTasksUseCase, apiKey string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !checkAdminKey(w, r, apiKey) {
			return
		}

		userID := r.PathValue("userId")
		if userID == "" {
			http.Error(w, "userId is required", http.StatusBadRequest)
			return
		}

		var req adminSetModerationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if err := uc.SetUserModeration(r.Context(), domain.UserID(userID), req.Enabled); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
