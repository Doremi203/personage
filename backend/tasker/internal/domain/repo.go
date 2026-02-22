package domain

import (
	"context"
	"time"
)

type EventRepo interface {
	UpsertEvent(ctx context.Context, event EventWithEmbedding) error
	GetEventsByClusterID(ctx context.Context, clusterID ClusterID) ([]Event, error)
}

type ClusterRepo interface {
	FindSimilarClusters(ctx context.Context, userID UserID, embedding []float32, topK int) ([]ClusterWithSimilarity, error)
	UpsertCluster(ctx context.Context, cluster Cluster) error
	FindClosableClusters(ctx context.Context, maxEventCount int, inactivityDuration time.Duration, limit int) ([]Cluster, error)
	UpdateClusterStatus(ctx context.Context, clusterID ClusterID, status ClusterStatus) error
}

type TaskRepo interface {
	CreateTask(ctx context.Context, task Task) error
	GetTasksByUserID(ctx context.Context, userID UserID) ([]Task, error)
	GetTasksByStatus(ctx context.Context, userID UserID, status TaskStatus) ([]Task, error)
	UpdateTaskSchedule(ctx context.Context, taskID TaskID, startTime time.Time, status TaskStatus) error
	UpdateTaskStatus(ctx context.Context, taskID TaskID, status TaskStatus) error
}
