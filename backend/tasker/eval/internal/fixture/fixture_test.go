package fixture_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Doremi203/personage/backend/tasker/eval/internal/fixture"
)

func writeFixture(t *testing.T, f fixture.Fixture) string {
	t.Helper()
	data, err := json.Marshal(f)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "fixture.json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func validFixture() fixture.Fixture {
	return fixture.Fixture{
		Version:    1,
		SnapshotID: "11111111-0000-0000-0000-000000000000",
		UserID:     "22222222-0000-0000-0000-000000000000",
		ExpectedTasks: []fixture.ExpectedTask{
			{
				Title:           "Review budget",
				Description:     "Check the spreadsheet",
				DurationMinutes: 30,
				Priority:        7,
				Category:        "work",
			},
		},
	}
}

func TestLoad_Valid(t *testing.T) {
	path := writeFixture(t, validFixture())
	if _, err := fixture.Load(path); err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
}

func TestLoad_InvalidVersion(t *testing.T) {
	f := validFixture()
	f.Version = 99
	path := writeFixture(t, f)
	if _, err := fixture.Load(path); err == nil {
		t.Error("expected error for wrong version")
	}
}

func TestLoad_MissingSnapshotID(t *testing.T) {
	f := validFixture()
	f.SnapshotID = ""
	path := writeFixture(t, f)
	if _, err := fixture.Load(path); err == nil {
		t.Error("expected error for missing snapshot_id")
	}
}

func TestLoad_InvalidCategory(t *testing.T) {
	f := validFixture()
	f.ExpectedTasks[0].Category = "hobbies"
	path := writeFixture(t, f)
	if _, err := fixture.Load(path); err == nil {
		t.Error("expected error for invalid category")
	}
}

func TestLoad_PriorityOutOfRange(t *testing.T) {
	f := validFixture()
	f.ExpectedTasks[0].Priority = 11
	path := writeFixture(t, f)
	if _, err := fixture.Load(path); err == nil {
		t.Error("expected error for priority > 10")
	}
}

func TestLoad_ZeroDuration(t *testing.T) {
	f := validFixture()
	f.ExpectedTasks[0].DurationMinutes = 0
	path := writeFixture(t, f)
	if _, err := fixture.Load(path); err == nil {
		t.Error("expected error for duration_minutes = 0")
	}
}

func TestLoad_EmptyExpectedTasks(t *testing.T) {
	f := validFixture()
	f.ExpectedTasks = nil
	path := writeFixture(t, f)
	if _, err := fixture.Load(path); err != nil {
		t.Errorf("Load returned error: %v", err)
	}
}

func TestLoad_RoundTrip(t *testing.T) {
	deadline := time.Date(2026, 4, 18, 23, 59, 59, 0, time.UTC)
	f := validFixture()
	f.ExpectedTasks[0].Deadline = &deadline

	path := writeFixture(t, f)
	got, err := fixture.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if got.SnapshotID != f.SnapshotID || got.UserID != f.UserID {
		t.Errorf("IDs mismatch: got snapshotID=%q userID=%q", got.SnapshotID, got.UserID)
	}
	if got.ExpectedTasks[0].Title != f.ExpectedTasks[0].Title {
		t.Errorf("title mismatch: got %q, want %q", got.ExpectedTasks[0].Title, f.ExpectedTasks[0].Title)
	}
	if got.ExpectedTasks[0].Deadline == nil || !got.ExpectedTasks[0].Deadline.Equal(deadline) {
		t.Errorf("deadline mismatch: got %v, want %v", got.ExpectedTasks[0].Deadline, deadline)
	}
}
