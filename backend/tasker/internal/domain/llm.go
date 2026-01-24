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
}

type TaskGenerationService interface {
	GenerateTask(ctx context.Context, events []Event) (GeneratedTask, error)
}
