package domain

import "time"

type TaskID string

func (id TaskID) String() string {
	return string(id)
}

type TaskStatus string

const (
	TaskStatusUnplanned TaskStatus = "unplanned"
	TaskStatusPlanned   TaskStatus = "planned"
	TaskStatusCompleted TaskStatus = "completed"
)

type TaskCategory string

func (t TaskCategory) String() string {
	switch t {
	case TaskCategoryWork:
		return "Работа"
	case TaskCategoryStudy:
		return "Учеба"
	case TaskCategoryPersonal:
		return "Личное"
	default:
		return ""
	}
}

const (
	TaskCategoryWork     TaskCategory = "work"
	TaskCategoryStudy    TaskCategory = "study"
	TaskCategoryPersonal TaskCategory = "personal"
)

type TaskPriority string

const (
	TaskPriorityLow  TaskPriority = "low"
	TaskPriorityMid  TaskPriority = "mid"
	TaskPriorityHigh TaskPriority = "high"
)

// PriorityFromInt converts an integer priority (1-10) to a TaskPriority string.
// 1-3 → low, 4-7 → mid, 8-10 → high.
func PriorityFromInt(p int) TaskPriority {
	switch {
	case p >= 8:
		return TaskPriorityHigh
	case p >= 4:
		return TaskPriorityMid
	default:
		return TaskPriorityLow
	}
}

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
	EndTime     *time.Time
	Status      TaskStatus
	Category    TaskCategory
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

const TimeSlotSize = 15 * time.Minute

type PlannedTask struct {
	ID    TaskID
	Start time.Time
	End   time.Time
}

type Schedule struct {
	Planned     []PlannedTask
	Unscheduled []Task
}

type TaskFilter struct {
	UserID   UserID
	Status   *TaskStatus   // nil means "all"
	Category *TaskCategory // nil means "all"
	Text     string        // full-text search query
	From     *time.Time    // inclusive lower bound on created_at
	Till     *time.Time    // inclusive upper bound on created_at
}

// Pagination holds page-based pagination parameters.
type Pagination struct {
	Page     int
	PageSize int
}

// TaskUpdate holds optional fields for partial task updates.
// nil pointers mean "do not update this field".
type TaskUpdate struct {
	Title       *string
	Description *string
	StartTime   *time.Time
	EndTime     *time.Time
	Category    *TaskCategory
}
