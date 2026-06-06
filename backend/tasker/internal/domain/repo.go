package domain

import (
	"context"
	"time"

	"github.com/Doremi203/personage/backend/libs/go/errors"
)

var (
	ErrTaskNotFound     = errors.Error("task not found")
	ErrClusterNotFound  = errors.Error("cluster not found")
	ErrProcessingPaused = errors.Error("processing stopped")
)

type EventRepo interface {
	UpsertEvent(ctx context.Context, event EventWithEmbedding) error
	GetEventsByClusterID(ctx context.Context, clusterID ClusterID) ([]Event, error)
	DeleteEventsByClusterID(ctx context.Context, clusterID ClusterID) error
	MaxSimilarityByClusters(ctx context.Context, clusterIDs []ClusterID, embedding []float32) (map[ClusterID]float64, error)
}

type ClusterRepo interface {
	FindSimilarClusters(ctx context.Context, userID UserID, embedding []float32, topK int) ([]ClusterWithSimilarity, error)
	FindSimilarClosedClusters(ctx context.Context, userID UserID, embedding []float32, topK int) ([]ClusterWithSimilarity, error)
	UpsertCluster(ctx context.Context, cluster Cluster) error
	FindClosableClusters(ctx context.Context, maxEventCount int, inactivityDuration time.Duration, limit int) ([]Cluster, error)
	UpdateClusterStatus(ctx context.Context, clusterID ClusterID, status ClusterStatus) error
	FinalizeCluster(
		ctx context.Context,
		clusterID ClusterID,
		outcome ClusterGenerationOutcome,
		reason *string,
	) error
	ListGenerationDiagnosticsByUserID(ctx context.Context, userID UserID) ([]ClusterGenerationDiagnostic, error)
	ListAdminClustersByUserID(ctx context.Context, userID UserID, limit int) ([]AdminClusterListItem, error)
	GetAdminClusterByID(ctx context.Context, clusterID ClusterID) (AdminClusterListItem, error)
	DeleteCluster(ctx context.Context, clusterID ClusterID) error
	// RecoverStaleClusters resets clusters stuck in 'processing' for longer than
	// the given staleThreshold back to 'open' so they can be retried.
	// Returns the number of recovered clusters.
	RecoverStaleClusters(ctx context.Context, staleThreshold time.Duration) (int, error)
}

type PromptRepo interface {
	GetPrompt(ctx context.Context, id PromptID) (Prompt, error)
	ListPrompts(ctx context.Context) ([]Prompt, error)
	UpdatePrompt(ctx context.Context, id PromptID, update PromptUpdate) (Prompt, error)
}

//go:generate mockgen -source=repo.go -destination=mock/repo_mock.go -typed

type ProcessingPauseRepo interface {
	IsPaused(ctx context.Context, userID UserID) (bool, error)
}

type ManualModerationRepo interface {
	RequiresModeration(ctx context.Context, userID UserID) (bool, error)
	AddUser(ctx context.Context, userID UserID) error
	RemoveUser(ctx context.Context, userID UserID) error
	ListUsers(ctx context.Context) ([]UserID, error)
}

type TaskRepo interface {
	CreateTask(ctx context.Context, task Task) error
	GetTaskByID(ctx context.Context, taskID TaskID, userID UserID) (Task, error)
	GetTasksByUserID(ctx context.Context, userID UserID) ([]Task, error)
	GetTasksByStatus(ctx context.Context, userID UserID, status TaskStatus) ([]Task, error)
	GetPlannedTasksInRange(ctx context.Context, userID UserID, from time.Time, to time.Time) ([]Task, error)
	GetUsersWithUnplannedTasks(ctx context.Context) ([]UserID, error)
	GetUsersWithPlannedTasks(ctx context.Context) ([]UserID, error)
	UpdateTaskSchedule(ctx context.Context, taskID TaskID, startTime time.Time, endTime time.Time, status TaskStatus) error
	UpdateTaskStatus(ctx context.Context, taskID TaskID, status TaskStatus) error
	UpdateTask(ctx context.Context, taskID TaskID, userID UserID, update TaskUpdate) (Task, error)
	DeleteTask(ctx context.Context, taskID TaskID) error
	ListTasks(ctx context.Context, filter TaskFilter, pagination Pagination) ([]Task, int, error)
}
