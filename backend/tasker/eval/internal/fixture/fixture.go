package fixture

import (
	"encoding/json"
	"os"
	"time"

	"github.com/Doremi203/personage/backend/libs/go/errors"
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
	ID              string     `json:"id,omitzero"`
	Title           string     `json:"title"`
	Description     string     `json:"description"`
	DurationMinutes int        `json:"duration_minutes"`
	Priority        int        `json:"priority"`
	Deadline        *time.Time `json:"deadline,omitzero"`
	StartTime       *time.Time `json:"start_time,omitzero"`
	Category        string     `json:"category"`
}

func (et ExpectedTask) ToScoreTask() score.Task {
	return score.Task{
		ID:              et.ID,
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
		return Fixture{}, errors.WrapFailf(err, "read fixture %v", errors.Token("path", path))
	}

	var fix Fixture
	if err := json.Unmarshal(data, &fix); err != nil {
		return Fixture{}, errors.WrapFailf(err, "parse fixture %v", errors.Token("path", path))
	}

	if err := validate(fix); err != nil {
		return Fixture{}, errors.WrapFailf(err, "invalid fixture %v", errors.Token("path", path))
	}

	return fix, nil
}

var validCategories = map[string]bool{
	"work": true, "study": true, "personal": true,
}

func validate(f Fixture) error {
	if f.Version != currentVersion {
		return errors.Errorf(
			"unsupported version %v (want %v)",
			errors.Token("version", f.Version),
			errors.Token("expected_version", currentVersion),
		)
	}
	if f.SnapshotID == "" {
		return errors.Errorf("snapshot_id is required")
	}
	if f.UserID == "" {
		return errors.Errorf("user_id is required")
	}
	for i, et := range f.ExpectedTasks {
		if et.Title == "" {
			return errors.Errorf("expected_tasks[%v].title is required", errors.Token("index", i))
		}
		if !validCategories[et.Category] {
			return errors.Errorf(
				"expected_tasks[%v].category %v must be work|study|personal",
				errors.Token("index", i),
				errors.Token("category", et.Category),
			)
		}
		if et.Priority < 1 || et.Priority > 10 {
			return errors.Errorf(
				"expected_tasks[%v].priority %v must be 1-10",
				errors.Token("index", i),
				errors.Token("priority", et.Priority),
			)
		}
		if et.DurationMinutes <= 0 {
			return errors.Errorf(
				"expected_tasks[%v].duration_minutes must be > 0",
				errors.Token("index", i),
			)
		}
	}
	return nil
}
