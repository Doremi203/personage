package scheduler

import (
	"testing"
	"time"

	"github.com/Doremi203/personage/backend/tasker/internal/domain"
)

func TestSchedule_EmptyTasks(t *testing.T) {
	start := time.Date(2026, 1, 22, 9, 0, 0, 0, time.UTC)
	window := 2 * time.Hour

	result := CalculateSchedule([]domain.Task{}, start, window)

	if len(result.Planned) != 0 {
		t.Errorf("Expected 0 planned tasks, got %d", len(result.Planned))
	}
	if len(result.Unscheduled) != 0 {
		t.Errorf("Expected 0 unscheduled tasks, got %d", len(result.Unscheduled))
	}
}

func TestSchedule_SingleFlexibleTask(t *testing.T) {
	start := time.Date(2026, 1, 22, 9, 0, 0, 0, time.UTC)
	window := 2 * time.Hour

	tasks := []domain.Task{
		{
			ID:       "task1",
			Duration: 30 * time.Minute,
			Priority: 1,
		},
	}

	result := CalculateSchedule(tasks, start, window)

	if len(result.Planned) != 1 {
		t.Fatalf("Expected 1 planned task, got %d", len(result.Planned))
	}
	if len(result.Unscheduled) != 0 {
		t.Errorf("Expected 0 unscheduled tasks, got %d", len(result.Unscheduled))
	}

	planned := result.Planned[0]
	if planned.ID != "task1" {
		t.Errorf("Expected task1, got %s", planned.ID)
	}
	if !planned.Start.Equal(start) {
		t.Errorf("Expected start at %v, got %v", start, planned.Start)
	}
	expectedEnd := start.Add(30 * time.Minute)
	if !planned.End.Equal(expectedEnd) {
		t.Errorf("Expected end at %v, got %v", expectedEnd, planned.End)
	}
}

func TestSchedule_SingleFixedTask(t *testing.T) {
	start := time.Date(2026, 1, 22, 9, 0, 0, 0, time.UTC)
	window := 2 * time.Hour
	fixedStart := time.Date(2026, 1, 22, 9, 30, 0, 0, time.UTC)

	tasks := []domain.Task{
		{
			ID:        "fixed1",
			Duration:  20 * time.Minute,
			StartTime: &fixedStart,
			Priority:  1,
		},
	}

	result := CalculateSchedule(tasks, start, window)

	if len(result.Planned) != 1 {
		t.Fatalf("Expected 1 planned task, got %d", len(result.Planned))
	}
	if len(result.Unscheduled) != 0 {
		t.Errorf("Expected 0 unscheduled tasks, got %d", len(result.Unscheduled))
	}

	planned := result.Planned[0]
	if planned.ID != "fixed1" {
		t.Errorf("Expected fixed1, got %s", planned.ID)
	}
	if !planned.Start.Equal(fixedStart) {
		t.Errorf("Expected start at %v, got %v", fixedStart, planned.Start)
	}
}

func TestSchedule_FlexibleTaskAroundFixed(t *testing.T) {
	start := time.Date(2026, 1, 22, 9, 0, 0, 0, time.UTC)
	window := 2 * time.Hour
	fixedStart := time.Date(2026, 1, 22, 9, 30, 0, 0, time.UTC)

	tasks := []domain.Task{
		{
			ID:        "fixed1",
			Duration:  30 * time.Minute,
			StartTime: &fixedStart,
			Priority:  1,
		},
		{
			ID:       "flex1",
			Duration: 20 * time.Minute,
			Priority: 1,
		},
		{
			ID:       "flex2",
			Duration: 15 * time.Minute,
			Priority: 1,
		},
	}

	result := CalculateSchedule(tasks, start, window)

	if len(result.Planned) != 3 {
		t.Fatalf("Expected 3 planned tasks, got %d", len(result.Planned))
	}
	if len(result.Unscheduled) != 0 {
		t.Errorf("Expected 0 unscheduled tasks, got %d", len(result.Unscheduled))
	}

	// flex1 should be scheduled before fixed task (9:00-9:20)
	flex1 := findPlannedTask(result.Planned, "flex1")
	if flex1 == nil {
		t.Fatal("flex1 not found in planned tasks")
	}
	expectedStart := start
	if !flex1.Start.Equal(expectedStart) {
		t.Errorf("flex1: Expected start at %v, got %v", expectedStart, flex1.Start)
	}

	// flex2 should be scheduled after fixed task (10:00-10:15)
	flex2 := findPlannedTask(result.Planned, "flex2")
	if flex2 == nil {
		t.Fatal("flex2 not found in planned tasks")
	}
	expectedStart2 := time.Date(2026, 1, 22, 10, 0, 0, 0, time.UTC)
	if !flex2.Start.Equal(expectedStart2) {
		t.Errorf("flex2: Expected start at %v, got %v", expectedStart2, flex2.Start)
	}
}

func TestSchedule_PrioritySorting(t *testing.T) {
	start := time.Date(2026, 1, 22, 9, 0, 0, 0, time.UTC)
	window := 1 * time.Hour

	tasks := []domain.Task{
		{
			ID:       "low",
			Duration: 15 * time.Minute,
			Priority: 1,
		},
		{
			ID:       "high",
			Duration: 15 * time.Minute,
			Priority: 10,
		},
		{
			ID:       "medium",
			Duration: 15 * time.Minute,
			Priority: 5,
		},
	}

	result := CalculateSchedule(tasks, start, window)

	if len(result.Planned) != 3 {
		t.Fatalf("Expected 3 planned tasks, got %d", len(result.Planned))
	}

	// High priority should be scheduled first
	high := findPlannedTask(result.Planned, "high")
	if high == nil {
		t.Fatal("high priority task not found")
	}
	if !high.Start.Equal(start) {
		t.Errorf("High priority task should start first at %v, got %v", start, high.Start)
	}

	// Medium priority should be second
	medium := findPlannedTask(result.Planned, "medium")
	if medium == nil {
		t.Fatal("medium priority task not found")
	}
	expectedMediumStart := start.Add(15 * time.Minute)
	if !medium.Start.Equal(expectedMediumStart) {
		t.Errorf("Medium priority task should start at %v, got %v", expectedMediumStart, medium.Start)
	}

	// Low priority should be last
	low := findPlannedTask(result.Planned, "low")
	if low == nil {
		t.Fatal("low priority task not found")
	}
	expectedLowStart := start.Add(30 * time.Minute)
	if !low.Start.Equal(expectedLowStart) {
		t.Errorf("Low priority task should start at %v, got %v", expectedLowStart, low.Start)
	}
}

func TestSchedule_DeadlineSorting(t *testing.T) {
	start := time.Date(2026, 1, 22, 9, 0, 0, 0, time.UTC)
	window := 2 * time.Hour

	deadline1 := time.Date(2026, 1, 22, 9, 30, 0, 0, time.UTC)
	deadline2 := time.Date(2026, 1, 22, 10, 0, 0, 0, time.UTC)

	tasks := []domain.Task{
		{
			ID:       "no_deadline",
			Duration: 15 * time.Minute,
			Priority: 5,
		},
		{
			ID:       "far_deadline",
			Duration: 15 * time.Minute,
			Priority: 5,
			Deadline: &deadline2,
		},
		{
			ID:       "near_deadline",
			Duration: 15 * time.Minute,
			Priority: 5,
			Deadline: &deadline1,
		},
	}

	result := CalculateSchedule(tasks, start, window)

	if len(result.Planned) != 3 {
		t.Fatalf("Expected 3 planned tasks, got %d", len(result.Planned))
	}

	// Near deadline should be first
	near := findPlannedTask(result.Planned, "near_deadline")
	if near == nil {
		t.Fatal("near_deadline task not found")
	}
	if !near.Start.Equal(start) {
		t.Errorf("Near deadline task should start first at %v, got %v", start, near.Start)
	}

	// Far deadline should be second
	far := findPlannedTask(result.Planned, "far_deadline")
	if far == nil {
		t.Fatal("far_deadline task not found")
	}
	if !far.Start.After(near.Start) {
		t.Errorf("Far deadline task should start after near deadline")
	}

	// No deadline should be last
	noDeadline := findPlannedTask(result.Planned, "no_deadline")
	if noDeadline == nil {
		t.Fatal("no_deadline task not found")
	}
	if !noDeadline.Start.After(far.Start) {
		t.Errorf("No deadline task should start after tasks with deadlines")
	}
}

func TestSchedule_DurationSorting(t *testing.T) {
	start := time.Date(2026, 1, 22, 9, 0, 0, 0, time.UTC)
	window := 2 * time.Hour

	tasks := []domain.Task{
		{
			ID:       "short",
			Duration: 10 * time.Minute,
			Priority: 5,
		},
		{
			ID:       "long",
			Duration: 40 * time.Minute,
			Priority: 5,
		},
		{
			ID:       "medium",
			Duration: 20 * time.Minute,
			Priority: 5,
		},
	}

	result := CalculateSchedule(tasks, start, window)

	if len(result.Planned) != 3 {
		t.Fatalf("Expected 3 planned tasks, got %d", len(result.Planned))
	}

	// Long duration should be scheduled first (harder to fit)
	long := findPlannedTask(result.Planned, "long")
	if long == nil {
		t.Fatal("long task not found")
	}
	if !long.Start.Equal(start) {
		t.Errorf("Long task should start first at %v, got %v", start, long.Start)
	}
}

func TestSchedule_DeadlineConstraint(t *testing.T) {
	start := time.Date(2026, 1, 22, 9, 0, 0, 0, time.UTC)
	window := 2 * time.Hour
	deadline := time.Date(2026, 1, 22, 9, 25, 0, 0, time.UTC)

	tasks := []domain.Task{
		{
			ID:       "task1",
			Duration: 30 * time.Minute,
			Priority: 1,
			Deadline: &deadline,
		},
	}

	result := CalculateSchedule(tasks, start, window)

	// Task requires 30 minutes but deadline is at 9:25, so it cannot be scheduled
	if len(result.Planned) != 0 {
		t.Errorf("Expected 0 planned tasks (deadline too tight), got %d", len(result.Planned))
	}
	if len(result.Unscheduled) != 1 {
		t.Fatalf("Expected 1 unscheduled task, got %d", len(result.Unscheduled))
	}
	if result.Unscheduled[0].ID != "task1" {
		t.Errorf("Expected task1 to be unscheduled, got %s", result.Unscheduled[0].ID)
	}
}

func TestSchedule_DeadlineRespected(t *testing.T) {
	start := time.Date(2026, 1, 22, 9, 0, 0, 0, time.UTC)
	window := 2 * time.Hour
	fixedStart := time.Date(2026, 1, 22, 9, 0, 0, 0, time.UTC)
	deadline := time.Date(2026, 1, 22, 9, 50, 0, 0, time.UTC)

	tasks := []domain.Task{
		{
			ID:        "fixed",
			Duration:  15 * time.Minute,
			StartTime: &fixedStart,
			Priority:  1,
		},
		{
			ID:       "with_deadline",
			Duration: 15 * time.Minute,
			Priority: 1,
			Deadline: &deadline,
		},
	}

	result := CalculateSchedule(tasks, start, window)

	if len(result.Planned) != 2 {
		t.Fatalf("Expected 2 planned tasks, got %d", len(result.Planned))
	}

	// Task with deadline should be scheduled after fixed task
	withDeadline := findPlannedTask(result.Planned, "with_deadline")
	if withDeadline == nil {
		t.Fatal("with_deadline task not found")
	}

	// Should start at 9:15 and end at 9:30, which is before the 9:50 deadline
	expectedStart := time.Date(2026, 1, 22, 9, 15, 0, 0, time.UTC)
	if !withDeadline.Start.Equal(expectedStart) {
		t.Errorf("Expected start at %v, got %v", expectedStart, withDeadline.Start)
	}
	if !withDeadline.End.Before(deadline) {
		t.Errorf("Task end %v should be before deadline %v", withDeadline.End, deadline)
	}
}

func TestSchedule_NoSpaceAvailable(t *testing.T) {
	start := time.Date(2026, 1, 22, 9, 0, 0, 0, time.UTC)
	window := 1 * time.Hour
	fixedStart := time.Date(2026, 1, 22, 9, 0, 0, 0, time.UTC)

	tasks := []domain.Task{
		{
			ID:        "fixed",
			Duration:  1 * time.Hour,
			StartTime: &fixedStart,
			Priority:  1,
		},
		{
			ID:       "flex",
			Duration: 10 * time.Minute,
			Priority: 1,
		},
	}

	result := CalculateSchedule(tasks, start, window)

	if len(result.Planned) != 1 {
		t.Errorf("Expected 1 planned task (only fixed), got %d", len(result.Planned))
	}
	if len(result.Unscheduled) != 1 {
		t.Fatalf("Expected 1 unscheduled task, got %d", len(result.Unscheduled))
	}
	if result.Unscheduled[0].ID != "flex" {
		t.Errorf("Expected flex to be unscheduled, got %s", result.Unscheduled[0].ID)
	}
}

func TestSchedule_PartiallyOutsideWindow_StartBefore(t *testing.T) {
	start := time.Date(2026, 1, 22, 9, 0, 0, 0, time.UTC)
	window := 1 * time.Hour
	fixedStart := time.Date(2026, 1, 22, 8, 50, 0, 0, time.UTC) // Starts 10 min before window

	tasks := []domain.Task{
		{
			ID:        "fixed",
			Duration:  30 * time.Minute, // Ends at 9:20
			StartTime: &fixedStart,
			Priority:  1,
		},
		{
			ID:       "flex",
			Duration: 15 * time.Minute,
			Priority: 1,
		},
	}

	result := CalculateSchedule(tasks, start, window)

	if len(result.Planned) != 2 {
		t.Fatalf("Expected 2 planned tasks, got %d", len(result.Planned))
	}

	// Flexible task should be scheduled after the fixed task ends (9:30, next 15-min slot after 9:20)
	flex := findPlannedTask(result.Planned, "flex")
	if flex == nil {
		t.Fatal("flex task not found")
	}
	expectedStart := time.Date(2026, 1, 22, 9, 30, 0, 0, time.UTC)
	if !flex.Start.Equal(expectedStart) {
		t.Errorf("Expected flex to start at %v, got %v", expectedStart, flex.Start)
	}
}

func TestSchedule_PartiallyOutsideWindow_EndAfter(t *testing.T) {
	start := time.Date(2026, 1, 22, 9, 0, 0, 0, time.UTC)
	window := 1 * time.Hour
	fixedStart := time.Date(2026, 1, 22, 9, 50, 0, 0, time.UTC) // Starts near end of window

	tasks := []domain.Task{
		{
			ID:        "fixed",
			Duration:  30 * time.Minute, // Ends at 10:20, beyond window
			StartTime: &fixedStart,
			Priority:  1,
		},
		{
			ID:       "flex",
			Duration: 10 * time.Minute,
			Priority: 1,
		},
	}

	result := CalculateSchedule(tasks, start, window)

	if len(result.Planned) != 2 {
		t.Fatalf("Expected 2 planned tasks, got %d", len(result.Planned))
	}

	// Flexible task should be scheduled before the fixed task
	flex := findPlannedTask(result.Planned, "flex")
	if flex == nil {
		t.Fatal("flex task not found")
	}
	if !flex.Start.Equal(start) {
		t.Errorf("Expected flex to start at %v, got %v", start, flex.Start)
	}
}

func TestSchedule_DurationRoundingUp(t *testing.T) {
	start := time.Date(2026, 1, 22, 9, 0, 0, 0, time.UTC)
	window := 1 * time.Hour

	tasks := []domain.Task{
		{
			ID:       "task1",
			Duration: 20 * time.Minute, // Should round up to 2 slots (30 minutes)
			Priority: 1,
		},
		{
			ID:       "task2",
			Duration: 5 * time.Minute, // Should round up to 1 slot (15 minutes)
			Priority: 1,
		},
	}

	result := CalculateSchedule(tasks, start, window)

	if len(result.Planned) != 2 {
		t.Fatalf("Expected 2 planned tasks, got %d", len(result.Planned))
	}

	task1 := findPlannedTask(result.Planned, "task1")
	task2 := findPlannedTask(result.Planned, "task2")

	if task1 == nil || task2 == nil {
		t.Fatal("Tasks not found in planned")
	}

	// task1 starts at 9:00, task2 should start at 9:30 (after 2 slots)
	expectedTask2Start := time.Date(2026, 1, 22, 9, 30, 0, 0, time.UTC)
	if !task2.Start.Equal(expectedTask2Start) {
		t.Errorf("Expected task2 to start at %v (after rounded task1), got %v", expectedTask2Start, task2.Start)
	}

	// task1 (20 min) rounds up to 2 slots → end must be slot-aligned at 9:30, not 9:20
	expectedTask1End := time.Date(2026, 1, 22, 9, 30, 0, 0, time.UTC)
	if !task1.End.Equal(expectedTask1End) {
		t.Errorf("Expected task1 end at %v (slot-aligned), got %v", expectedTask1End, task1.End)
	}

	// task2 (5 min) rounds up to 1 slot → end must be slot-aligned at 9:45, not 9:35
	expectedTask2End := time.Date(2026, 1, 22, 9, 45, 0, 0, time.UTC)
	if !task2.End.Equal(expectedTask2End) {
		t.Errorf("Expected task2 end at %v (slot-aligned), got %v", expectedTask2End, task2.End)
	}
}

// TestSchedule_SlotAlignedEndTime verifies that PlannedTask.End is always aligned to a
// 15-minute slot boundary, even when the raw task duration is not a multiple of 15 minutes.
func TestSchedule_SlotAlignedEndTime(t *testing.T) {
	start := time.Date(2026, 1, 22, 9, 0, 0, 0, time.UTC)
	window := 2 * time.Hour

	tests := []struct {
		name        string
		duration    time.Duration
		expectedEnd time.Time
	}{
		// 33 min → ceil(33/15) = 3 slots → end = 9:00 + 45 min
		{"33min_task", 33 * time.Minute, start.Add(45 * time.Minute)},
		// 28 min → ceil(28/15) = 2 slots → end = 9:00 + 30 min
		{"28min_task", 28 * time.Minute, start.Add(30 * time.Minute)},
		// 1 min → ceil(1/15) = 1 slot → end = 9:00 + 15 min
		{"1min_task", 1 * time.Minute, start.Add(15 * time.Minute)},
		// 15 min → ceil(15/15) = 1 slot → end = 9:00 + 15 min
		{"15min_task", 15 * time.Minute, start.Add(15 * time.Minute)},
		// 30 min → ceil(30/15) = 2 slots → end = 9:00 + 30 min
		{"30min_task", 30 * time.Minute, start.Add(30 * time.Minute)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateSchedule([]domain.Task{
				{ID: domain.TaskID(tt.name), Duration: tt.duration, Priority: 1},
			}, start, window)

			if len(result.Planned) != 1 {
				t.Fatalf("Expected 1 planned task, got %d", len(result.Planned))
			}
			task := result.Planned[0]
			if !task.End.Equal(tt.expectedEnd) {
				t.Errorf("duration=%v: expected slot-aligned end %v, got %v", tt.duration, tt.expectedEnd, task.End)
			}
		})
	}
}

// TestSchedule_FixedTask_SlotAlignedEndTime verifies that fixed tasks also get
// slot-aligned end times stored in PlannedTask.
func TestSchedule_FixedTask_SlotAlignedEndTime(t *testing.T) {
	start := time.Date(2026, 1, 22, 9, 0, 0, 0, time.UTC)
	window := 2 * time.Hour
	fixedStart := time.Date(2026, 1, 22, 9, 30, 0, 0, time.UTC)

	tasks := []domain.Task{
		{
			ID:        "fixed1",
			Duration:  20 * time.Minute, // 20 min → ceil(20/15) = 2 slots → end at 9:30 + 30 min = 10:00
			StartTime: &fixedStart,
			Priority:  1,
		},
	}

	result := CalculateSchedule(tasks, start, window)

	if len(result.Planned) != 1 {
		t.Fatalf("Expected 1 planned task, got %d", len(result.Planned))
	}

	planned := result.Planned[0]
	expectedEnd := time.Date(2026, 1, 22, 10, 0, 0, 0, time.UTC)
	if !planned.End.Equal(expectedEnd) {
		t.Errorf("Expected slot-aligned end %v, got %v", expectedEnd, planned.End)
	}
}

func TestSchedule_ExactSlotBoundary(t *testing.T) {
	start := time.Date(2026, 1, 22, 9, 0, 0, 0, time.UTC)
	window := 1 * time.Hour
	fixedStart := time.Date(2026, 1, 22, 9, 0, 0, 0, time.UTC)

	tasks := []domain.Task{
		{
			ID:        "fixed",
			Duration:  30 * time.Minute, // Exactly 2 slots
			StartTime: &fixedStart,
			Priority:  1,
		},
		{
			ID:       "flex",
			Duration: 15 * time.Minute,
			Priority: 1,
		},
	}

	result := CalculateSchedule(tasks, start, window)

	if len(result.Planned) != 2 {
		t.Fatalf("Expected 2 planned tasks, got %d", len(result.Planned))
	}

	flex := findPlannedTask(result.Planned, "flex")
	if flex == nil {
		t.Fatal("flex task not found")
	}

	// Should start exactly at 9:30
	expectedStart := time.Date(2026, 1, 22, 9, 30, 0, 0, time.UTC)
	if !flex.Start.Equal(expectedStart) {
		t.Errorf("Expected flex to start at %v, got %v", expectedStart, flex.Start)
	}
}

func TestSchedule_MultipleUnscheduledTasks(t *testing.T) {
	start := time.Date(2026, 1, 22, 9, 0, 0, 0, time.UTC)
	window := 30 * time.Minute

	tasks := []domain.Task{
		{
			ID:       "task1",
			Duration: 15 * time.Minute,
			Priority: 1,
		},
		{
			ID:       "task2",
			Duration: 15 * time.Minute,
			Priority: 1,
		},
		{
			ID:       "task3",
			Duration: 15 * time.Minute,
			Priority: 1,
		},
	}

	result := CalculateSchedule(tasks, start, window)

	// Only 2 tasks can fit in 30 minutes (2 slots)
	if len(result.Planned) != 2 {
		t.Errorf("Expected 2 planned tasks, got %d", len(result.Planned))
	}
	if len(result.Unscheduled) != 1 {
		t.Fatalf("Expected 1 unscheduled task, got %d", len(result.Unscheduled))
	}
}

func TestSchedule_DeadlineBeyondWindow(t *testing.T) {
	start := time.Date(2026, 1, 22, 9, 0, 0, 0, time.UTC)
	window := 1 * time.Hour
	deadline := time.Date(2026, 1, 22, 12, 0, 0, 0, time.UTC) // Way beyond window

	tasks := []domain.Task{
		{
			ID:       "task1",
			Duration: 20 * time.Minute,
			Priority: 1,
			Deadline: &deadline,
		},
	}

	result := CalculateSchedule(tasks, start, window)

	// Should be scheduled normally since deadline is beyond window
	if len(result.Planned) != 1 {
		t.Fatalf("Expected 1 planned task, got %d", len(result.Planned))
	}
	if len(result.Unscheduled) != 0 {
		t.Errorf("Expected 0 unscheduled tasks, got %d", len(result.Unscheduled))
	}
}

func TestSchedule_ComplexScenario(t *testing.T) {
	start := time.Date(2026, 1, 22, 9, 0, 0, 0, time.UTC)
	window := 3 * time.Hour

	fixedStart1 := time.Date(2026, 1, 22, 9, 30, 0, 0, time.UTC)
	fixedStart2 := time.Date(2026, 1, 22, 10, 30, 0, 0, time.UTC)
	deadline1 := time.Date(2026, 1, 22, 9, 40, 0, 0, time.UTC)
	deadline2 := time.Date(2026, 1, 22, 11, 30, 0, 0, time.UTC)

	tasks := []domain.Task{
		// Fixed tasks
		{
			ID:        "meeting1",
			Duration:  30 * time.Minute,
			StartTime: &fixedStart1,
			Priority:  1,
		},
		{
			ID:        "meeting2",
			Duration:  20 * time.Minute,
			StartTime: &fixedStart2,
			Priority:  1,
		},
		// Flexible tasks
		{
			ID:       "urgent",
			Duration: 15 * time.Minute,
			Priority: 10,
			Deadline: &deadline1,
		},
		{
			ID:       "important",
			Duration: 25 * time.Minute,
			Priority: 8,
			Deadline: &deadline2,
		},
		{
			ID:       "normal",
			Duration: 20 * time.Minute,
			Priority: 5,
		},
		{
			ID:       "low",
			Duration: 30 * time.Minute,
			Priority: 1,
		},
	}

	result := CalculateSchedule(tasks, start, window)

	// All tasks should be schedulable
	if len(result.Planned) < 4 {
		t.Errorf("Expected at least 4 planned tasks, got %d", len(result.Planned))
	}

	// Verify urgent task is scheduled before its deadline
	urgent := findPlannedTask(result.Planned, "urgent")
	if urgent != nil {
		if !urgent.End.Before(deadline1) && !urgent.End.Equal(deadline1) {
			t.Errorf("Urgent task end %v should be before or at deadline %v", urgent.End, deadline1)
		}
	}

	// Verify important task is scheduled before its deadline
	important := findPlannedTask(result.Planned, "important")
	if important != nil {
		if !important.End.Before(deadline2) && !important.End.Equal(deadline2) {
			t.Errorf("Important task end %v should be before or at deadline %v", important.End, deadline2)
		}
	}
}

func TestSchedule_TableDriven(t *testing.T) {
	baseStart := time.Date(2026, 1, 22, 9, 0, 0, 0, time.UTC)

	tests := []struct {
		name                string
		tasks               []domain.Task
		planningStart       time.Time
		windowDuration      time.Duration
		expectedPlanned     int
		expectedUnscheduled int
		validate            func(t *testing.T, result domain.Schedule)
	}{
		{
			name: "Full day with meetings and tasks",
			tasks: []domain.Task{
				{
					ID:        "morning_meeting",
					Duration:  30 * time.Minute,
					StartTime: ptr(baseStart.Add(1 * time.Hour)),
					Priority:  5,
				},
				{
					ID:        "lunch",
					Duration:  1 * time.Hour,
					StartTime: ptr(baseStart.Add(4 * time.Hour)),
					Priority:  5,
				},
				{
					ID:       "urgent_task",
					Duration: 45 * time.Minute,
					Priority: 10,
					Deadline: ptr(baseStart.Add(2 * time.Hour)),
				},
				{
					ID:       "important_task",
					Duration: 1 * time.Hour,
					Priority: 8,
				},
				{
					ID:       "normal_task",
					Duration: 30 * time.Minute,
					Priority: 5,
				},
				{
					ID:       "low_priority",
					Duration: 15 * time.Minute,
					Priority: 1,
				},
			},
			planningStart:       baseStart,
			windowDuration:      8 * time.Hour,
			expectedPlanned:     6,
			expectedUnscheduled: 0,
			validate: func(t *testing.T, result domain.Schedule) {
				// Urgent task should be scheduled before its deadline
				urgent := findPlannedTask(result.Planned, "urgent_task")
				if urgent == nil {
					t.Fatal("urgent_task not found")
				}
				deadline := baseStart.Add(2 * time.Hour)
				if !urgent.End.Before(deadline) && !urgent.End.Equal(deadline) {
					t.Errorf("urgent_task end %v should be before deadline %v", urgent.End, deadline)
				}
			},
		},
		{
			name: "Overbooked schedule - some tasks cannot fit",
			tasks: []domain.Task{
				{
					ID:        "fixed1",
					Duration:  1 * time.Hour,
					StartTime: ptr(baseStart),
					Priority:  5,
				},
				{
					ID:        "fixed2",
					Duration:  1 * time.Hour,
					StartTime: ptr(baseStart.Add(2 * time.Hour)),
					Priority:  5,
				},
				{
					ID:       "flex1",
					Duration: 1 * time.Hour,
					Priority: 10,
				},
				{
					ID:       "flex2",
					Duration: 1 * time.Hour,
					Priority: 5,
				},
				{
					ID:       "flex3",
					Duration: 1 * time.Hour,
					Priority: 1,
				},
			},
			planningStart:       baseStart,
			windowDuration:      4 * time.Hour,
			expectedPlanned:     4,
			expectedUnscheduled: 1,
			validate: func(t *testing.T, result domain.Schedule) {
				// flex3 (lowest priority) should be unscheduled
				if len(result.Unscheduled) != 1 {
					t.Fatalf("Expected 1 unscheduled task, got %d", len(result.Unscheduled))
				}
				if result.Unscheduled[0].ID != "flex3" {
					t.Errorf("Expected flex3 to be unscheduled, got %s", result.Unscheduled[0].ID)
				}
			},
		},
		{
			name: "Priority-first sorting with same priority deadline matters",
			tasks: []domain.Task{
				{
					ID:       "task_far_deadline",
					Duration: 30 * time.Minute,
					Priority: 5,
					Deadline: ptr(baseStart.Add(3 * time.Hour)),
				},
				{
					ID:       "task_near_deadline",
					Duration: 30 * time.Minute,
					Priority: 5,
					Deadline: ptr(baseStart.Add(1 * time.Hour)),
				},
				{
					ID:       "task_medium_deadline",
					Duration: 30 * time.Minute,
					Priority: 5,
					Deadline: ptr(baseStart.Add(2 * time.Hour)),
				},
			},
			planningStart:       baseStart,
			windowDuration:      4 * time.Hour,
			expectedPlanned:     3,
			expectedUnscheduled: 0,
			validate: func(t *testing.T, result domain.Schedule) {
				// All tasks should be scheduled
				if len(result.Planned) != 3 {
					t.Errorf("Expected all 3 tasks to be scheduled, got %d", len(result.Planned))
				}

				// With same priority, near deadline task should be scheduled first
				nearDeadline := findPlannedTask(result.Planned, "task_near_deadline")
				farDeadline := findPlannedTask(result.Planned, "task_far_deadline")

				if nearDeadline != nil && farDeadline != nil {
					if !nearDeadline.Start.Before(farDeadline.Start) {
						t.Errorf("Near deadline task should start before far deadline task when priority is equal")
					}
				}
			},
		},
		{
			name: "Fragmented schedule with multiple gaps",
			tasks: []domain.Task{
				{
					ID:        "meeting1",
					Duration:  15 * time.Minute,
					StartTime: ptr(baseStart.Add(15 * time.Minute)),
					Priority:  5,
				},
				{
					ID:        "meeting2",
					Duration:  15 * time.Minute,
					StartTime: ptr(baseStart.Add(1 * time.Hour)),
					Priority:  5,
				},
				{
					ID:        "meeting3",
					Duration:  15 * time.Minute,
					StartTime: ptr(baseStart.Add(105 * time.Minute)),
					Priority:  5,
				},
				{
					ID:       "small_task",
					Duration: 15 * time.Minute,
					Priority: 5,
				},
				{
					ID:       "medium_task",
					Duration: 30 * time.Minute,
					Priority: 5,
				},
			},
			planningStart:       baseStart,
			windowDuration:      3 * time.Hour,
			expectedPlanned:     5,
			expectedUnscheduled: 0,
			validate: func(t *testing.T, result domain.Schedule) {
				// Both tasks should be scheduled in available gaps
				medium := findPlannedTask(result.Planned, "medium_task")
				small := findPlannedTask(result.Planned, "small_task")

				if medium == nil || small == nil {
					t.Fatal("Tasks not found")
				}

				// Verify both tasks are scheduled
				if medium.Start.IsZero() || small.Start.IsZero() {
					t.Errorf("Tasks should have valid start times")
				}
			},
		},
		{
			name: "Impossible deadlines - all tasks unscheduled",
			tasks: []domain.Task{
				{
					ID:       "task1",
					Duration: 1 * time.Hour,
					Priority: 10,
					Deadline: ptr(baseStart.Add(30 * time.Minute)),
				},
				{
					ID:       "task2",
					Duration: 45 * time.Minute,
					Priority: 5,
					Deadline: ptr(baseStart.Add(20 * time.Minute)),
				},
			},
			planningStart:       baseStart,
			windowDuration:      2 * time.Hour,
			expectedPlanned:     0,
			expectedUnscheduled: 2,
			validate: func(t *testing.T, result domain.Schedule) {
				if len(result.Unscheduled) != 2 {
					t.Errorf("Expected 2 unscheduled tasks, got %d", len(result.Unscheduled))
				}
			},
		},
		{
			name: "Mixed duration rounding scenarios",
			tasks: []domain.Task{
				{
					ID:       "task_1min",
					Duration: 1 * time.Minute, // Rounds to 1 slot (15 min)
					Priority: 5,
				},
				{
					ID:       "task_16min",
					Duration: 16 * time.Minute, // Rounds to 2 slots (30 min)
					Priority: 5,
				},
				{
					ID:       "task_29min",
					Duration: 29 * time.Minute, // Rounds to 2 slots (30 min)
					Priority: 5,
				},
				{
					ID:       "task_30min",
					Duration: 30 * time.Minute, // Exactly 2 slots
					Priority: 5,
				},
			},
			planningStart:       baseStart,
			windowDuration:      2 * time.Hour,
			expectedPlanned:     4,
			expectedUnscheduled: 0,
			validate: func(t *testing.T, result domain.Schedule) {
				// Verify all tasks are scheduled
				if len(result.Planned) != 4 {
					t.Errorf("Expected all 4 tasks to be scheduled")
				}

				// Verify tasks don't overlap (basic sanity check)
				for i := 0; i < len(result.Planned); i++ {
					for j := i + 1; j < len(result.Planned); j++ {
						task1 := result.Planned[i]
						task2 := result.Planned[j]

						// Check if tasks overlap
						if task1.Start.Before(task2.End) && task2.Start.Before(task1.End) {
							t.Errorf("Tasks %s and %s overlap", task1.ID, task2.ID)
						}
					}
				}
			},
		},
		{
			name: "Back-to-back fixed tasks with flexible tasks",
			tasks: []domain.Task{
				{
					ID:        "fixed1",
					Duration:  30 * time.Minute,
					StartTime: ptr(baseStart),
					Priority:  5,
				},
				{
					ID:        "fixed2",
					Duration:  30 * time.Minute,
					StartTime: ptr(baseStart.Add(30 * time.Minute)),
					Priority:  5,
				},
				{
					ID:        "fixed3",
					Duration:  30 * time.Minute,
					StartTime: ptr(baseStart.Add(60 * time.Minute)),
					Priority:  5,
				},
				{
					ID:       "flex1",
					Duration: 15 * time.Minute,
					Priority: 10,
				},
			},
			planningStart:       baseStart,
			windowDuration:      2 * time.Hour,
			expectedPlanned:     4,
			expectedUnscheduled: 0,
			validate: func(t *testing.T, result domain.Schedule) {
				flex := findPlannedTask(result.Planned, "flex1")
				if flex == nil {
					t.Fatal("flex1 not found")
				}

				// Should be scheduled after all fixed tasks (at 9:90 = 10:30)
				expectedStart := baseStart.Add(90 * time.Minute)
				if !flex.Start.Equal(expectedStart) {
					t.Errorf("Expected flex1 to start at %v, got %v", expectedStart, flex.Start)
				}
			},
		},
		{
			name: "Deadline exactly at window boundary",
			tasks: []domain.Task{
				{
					ID:       "task1",
					Duration: 30 * time.Minute,
					Priority: 5,
					Deadline: ptr(baseStart.Add(1 * time.Hour)), // Exactly at window end
				},
			},
			planningStart:       baseStart,
			windowDuration:      1 * time.Hour,
			expectedPlanned:     1,
			expectedUnscheduled: 0,
			validate: func(t *testing.T, result domain.Schedule) {
				task := findPlannedTask(result.Planned, "task1")
				if task == nil {
					t.Fatal("task1 not found")
				}

				deadline := baseStart.Add(1 * time.Hour)
				if task.End.After(deadline) {
					t.Errorf("Task end %v should not be after deadline %v", task.End, deadline)
				}
			},
		},
		{
			name: "Complex real-world scenario",
			tasks: []domain.Task{
				// Morning standup
				{
					ID:        "standup",
					Duration:  15 * time.Minute,
					StartTime: ptr(baseStart.Add(30 * time.Minute)),
					Priority:  8,
				},
				// Lunch break
				{
					ID:        "lunch",
					Duration:  1 * time.Hour,
					StartTime: ptr(baseStart.Add(4 * time.Hour)),
					Priority:  10,
				},
				// Afternoon meeting
				{
					ID:        "client_meeting",
					Duration:  1 * time.Hour,
					StartTime: ptr(baseStart.Add(6 * time.Hour)),
					Priority:  10,
				},
				// Urgent bug fix - must be done before standup
				{
					ID:       "critical_bug",
					Duration: 30 * time.Minute,
					Priority: 10,
					Deadline: ptr(baseStart.Add(30 * time.Minute)),
				},
				// Feature work
				{
					ID:       "feature_a",
					Duration: 2 * time.Hour,
					Priority: 7,
					Deadline: ptr(baseStart.Add(7 * time.Hour)),
				},
				{
					ID:       "feature_b",
					Duration: 1 * time.Hour,
					Priority: 6,
				},
				// Code review
				{
					ID:       "code_review",
					Duration: 30 * time.Minute,
					Priority: 5,
				},
				// Documentation
				{
					ID:       "docs",
					Duration: 45 * time.Minute,
					Priority: 3,
				},
			},
			planningStart:       baseStart,
			windowDuration:      8 * time.Hour,
			expectedPlanned:     8,
			expectedUnscheduled: 0,
			validate: func(t *testing.T, result domain.Schedule) {
				// Critical bug should be scheduled before standup
				bug := findPlannedTask(result.Planned, "critical_bug")
				standup := findPlannedTask(result.Planned, "standup")

				if bug == nil || standup == nil {
					t.Fatal("Critical tasks not found")
				}

				if !bug.End.Before(standup.Start) && !bug.End.Equal(standup.Start) {
					t.Errorf("Bug fix should complete before standup starts")
				}

				// Feature A should be completed before its deadline
				featureA := findPlannedTask(result.Planned, "feature_a")
				if featureA == nil {
					t.Fatal("feature_a not found")
				}

				deadline := baseStart.Add(7 * time.Hour)
				if featureA.End.After(deadline) {
					t.Errorf("feature_a end %v should be before deadline %v", featureA.End, deadline)
				}
			},
		},
		{
			name: "Zero duration window",
			tasks: []domain.Task{
				{
					ID:       "task1",
					Duration: 15 * time.Minute,
					Priority: 5,
				},
			},
			planningStart:       baseStart,
			windowDuration:      0,
			expectedPlanned:     0,
			expectedUnscheduled: 1,
			validate: func(t *testing.T, result domain.Schedule) {
				if len(result.Unscheduled) != 1 {
					t.Errorf("Expected 1 unscheduled task in zero window, got %d", len(result.Unscheduled))
				}
			},
		},
		{
			name: "Tasks with same priority, deadline, and duration",
			tasks: []domain.Task{
				{
					ID:       "task_a",
					Duration: 30 * time.Minute,
					Priority: 5,
					Deadline: ptr(baseStart.Add(2 * time.Hour)),
				},
				{
					ID:       "task_b",
					Duration: 30 * time.Minute,
					Priority: 5,
					Deadline: ptr(baseStart.Add(2 * time.Hour)),
				},
				{
					ID:       "task_c",
					Duration: 30 * time.Minute,
					Priority: 5,
					Deadline: ptr(baseStart.Add(2 * time.Hour)),
				},
			},
			planningStart:       baseStart,
			windowDuration:      2 * time.Hour,
			expectedPlanned:     3,
			expectedUnscheduled: 0,
			validate: func(t *testing.T, result domain.Schedule) {
				// All should be scheduled (order doesn't matter since they're identical)
				if len(result.Planned) != 3 {
					t.Errorf("Expected all 3 identical tasks to be scheduled")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateSchedule(tt.tasks, tt.planningStart, tt.windowDuration)

			if len(result.Planned) != tt.expectedPlanned {
				t.Errorf("Expected %d planned tasks, got %d", tt.expectedPlanned, len(result.Planned))
			}

			if len(result.Unscheduled) != tt.expectedUnscheduled {
				t.Errorf("Expected %d unscheduled tasks, got %d", tt.expectedUnscheduled, len(result.Unscheduled))
			}

			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

// Helper function to create time pointer
func ptr(t time.Time) *time.Time {
	return &t
}

// Helper function to find a planned task by ID
func findPlannedTask(planned []domain.PlannedTask, id domain.TaskID) *domain.PlannedTask {
	for i := range planned {
		if planned[i].ID == id {
			return &planned[i]
		}
	}
	return nil
}
