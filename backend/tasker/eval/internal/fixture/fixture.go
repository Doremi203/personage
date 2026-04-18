package fixture

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/Doremi203/personage/backend/tasker/eval/internal/score"
	"github.com/Doremi203/personage/backend/tasker/internal/domain"
)

const currentVersion = 1

// Fixture is the golden fixture file format.
type Fixture struct {
	Version       int            `json:"version"`
	SnapshotID    string         `json:"snapshot_id"`
	UserID        string         `json:"user_id"`
	ExpectedTasks []ExpectedTask `json:"expected_tasks"`
}

// ExpectedTask is a single hand-curated expected task.
type ExpectedTask struct {
	Title           string     `json:"title"`
	Description     string     `json:"description"`
	DurationMinutes int        `json:"duration_minutes"`
	Priority        int        `json:"priority"`
	Deadline        *time.Time `json:"deadline,omitempty"`
	StartTime       *time.Time `json:"start_time,omitempty"`
	Category        string     `json:"category"`
}

func (et ExpectedTask) ToScoreTask() score.Task {
	return score.Task{
		Title:           et.Title,
		Description:     et.Description,
		DurationMinutes: et.DurationMinutes,
		Priority:        et.Priority,
		Deadline:        et.Deadline,
		StartTime:       et.StartTime,
		Category:        domain.TaskCategory(et.Category),
	}
}

// Load reads and validates a fixture from path.
func Load(path string) (Fixture, error) {
	data, err := os.ReadFile(path) //#nosec G304 -- eval tool, operator-supplied path
	if err != nil {
		return Fixture{}, fmt.Errorf("read fixture %s: %w", path, err)
	}

	var fix Fixture
	if err := json.Unmarshal(data, &fix); err != nil {
		return Fixture{}, fmt.Errorf("parse fixture %s: %w", path, err)
	}

	if err := validate(fix); err != nil {
		return Fixture{}, fmt.Errorf("invalid fixture %s: %w", path, err)
	}

	return fix, nil
}

var validCategories = map[string]bool{
	"work": true, "study": true, "personal": true,
}

func validate(f Fixture) error {
	if f.Version != currentVersion {
		return fmt.Errorf("unsupported version %d (want %d)", f.Version, currentVersion)
	}
	if f.SnapshotID == "" {
		return fmt.Errorf("snapshot_id is required")
	}
	if f.UserID == "" {
		return fmt.Errorf("user_id is required")
	}
	if len(f.ExpectedTasks) == 0 {
		return fmt.Errorf("expected_tasks must not be empty")
	}
	for i, et := range f.ExpectedTasks {
		if et.Title == "" {
			return fmt.Errorf("expected_tasks[%d].title is required", i)
		}
		if !validCategories[et.Category] {
			return fmt.Errorf("expected_tasks[%d].category %q must be work|study|personal", i, et.Category)
		}
		if et.Priority < 1 || et.Priority > 10 {
			return fmt.Errorf("expected_tasks[%d].priority %d must be 1-10", i, et.Priority)
		}
		if et.DurationMinutes <= 0 {
			return fmt.Errorf("expected_tasks[%d].duration_minutes must be > 0", i)
		}
	}
	return nil
}
