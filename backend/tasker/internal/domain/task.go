package domain

import "time"

type TaskID string

func (id TaskID) String() string {
	return string(id)
}

type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusPlanned   TaskStatus = "planned"
	TaskStatusCompleted TaskStatus = "completed"
)

type Task struct {
	ID          TaskID
	UserID      UserID
	ClusterID   ClusterID
	Title       string
	Description string
	Duration    time.Duration
	Priority    int
	Deadline    *time.Time
	StartTime   *time.Time
	Status      TaskStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

const TimeSlotSize = 10 * time.Minute

type PlannedTask struct {
	ID    TaskID
	Start time.Time
	End   time.Time
}

type Schedule struct {
	Planned     []PlannedTask
	Unscheduled []Task
}
