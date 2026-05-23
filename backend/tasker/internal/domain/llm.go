package domain

import (
	"context"
	"time"
)

type GeneratedTask struct {
	Title           string
	Description     string
	DurationMinutes int
	Priority        int
	Deadline        *time.Time
	StartTime       *time.Time
	Category        string
}

type TaskGenerationDecision struct {
	ShouldGenerate bool
	Reason         *string
}

type ClusterClassificatorService interface {
	GetTaskGenerationDecision(ctx context.Context, events []Event, profile UserProfile) (TaskGenerationDecision, error)
}

type TaskGenerationService interface {
	GenerateTask(ctx context.Context, events []Event) (GeneratedTask, error)
}
