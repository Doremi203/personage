package scheduler

import (
	"math"
	"sort"
	"time"

	"github.com/Doremi203/personage/backend/tasker/internal/domain"
)

// CalculateSchedule takes a list of tasks and a planning window.
// It returns a Schedule containing both scheduled and unscheduled tasks.
// Tasks with fixed StartTime are included in the output as-is but occupy space in the grid.
func CalculateSchedule(tasks []domain.Task, planningStart time.Time, windowDuration time.Duration) domain.Schedule {
	// Calculate total number of slots in the planning window
	totalSlots := int(math.Ceil(float64(windowDuration) / float64(domain.TimeSlotSize)))

	// Initialize the grid - false means available, true means occupied
	grid := make([]bool, totalSlots)

	// Result slices
	var planned []domain.PlannedTask
	var unscheduled []domain.Task

	// Helper: Convert time to grid index
	timeToIndex := func(t time.Time) int {
		diff := t.Sub(planningStart)
		return int(diff / domain.TimeSlotSize)
	}

	// Helper: Convert grid index to time
	indexToTime := func(idx int) time.Time {
		return planningStart.Add(time.Duration(idx) * domain.TimeSlotSize)
	}

	// Helper: Convert duration to slot count (always round up)
	durationToSlots := func(d time.Duration) int {
		return int(math.Ceil(float64(d) / float64(domain.TimeSlotSize)))
	}

	// Phase 1: Process fixed tasks ("The Walls")
	for _, task := range tasks {
		if task.StartTime == nil {
			continue // Skip flexible tasks for now
		}

		// Calculate start and end indices
		startIdx := timeToIndex(*task.StartTime)
		endTime := task.StartTime.Add(task.Duration)
		endIdx := timeToIndex(endTime)

		// Handle tasks that end exactly on a slot boundary
		if endTime.Sub(indexToTime(endIdx)) == 0 && endIdx > 0 {
			// Task ends exactly at slot boundary, don't include that slot
		} else if endTime.Sub(indexToTime(endIdx)) > 0 {
			// Task extends into this slot, include it
			endIdx++
		}

		// Clamp to window boundaries
		if startIdx < 0 {
			startIdx = 0
		}
		if endIdx > totalSlots {
			endIdx = totalSlots
		}

		// Mark slots as occupied
		for i := startIdx; i < endIdx && i < totalSlots; i++ {
			if i >= 0 {
				grid[i] = true
			}
		}

		// Add to planned with slot-aligned end time
		planned = append(planned, domain.PlannedTask{
			ID:    task.ID,
			Start: *task.StartTime,
			End:   indexToTime(endIdx),
		})
	}

	// Phase 2: Sort flexible tasks ("The Sort")
	var flexibleTasks []domain.Task
	for _, task := range tasks {
		if task.StartTime == nil {
			flexibleTasks = append(flexibleTasks, task)
		}
	}

	sort.Slice(flexibleTasks, func(i, j int) bool {
		taskI, taskJ := flexibleTasks[i], flexibleTasks[j]

		// 1. Priority (Descending) - Higher priority first
		if taskI.Priority != taskJ.Priority {
			return taskI.Priority > taskJ.Priority
		}

		// 2. Deadline Tightness (Ascending) - Closer deadlines first
		// Tasks with deadlines come before tasks without deadlines
		hasDeadlineI := taskI.Deadline != nil
		hasDeadlineJ := taskJ.Deadline != nil

		if hasDeadlineI && !hasDeadlineJ {
			return true
		}
		if !hasDeadlineI && hasDeadlineJ {
			return false
		}
		if hasDeadlineI && hasDeadlineJ {
			if !taskI.Deadline.Equal(*taskJ.Deadline) {
				return taskI.Deadline.Before(*taskJ.Deadline)
			}
		}

		// 3. Duration (Descending) - Harder-to-fit (longer) tasks first
		return taskI.Duration > taskJ.Duration
	})

	// Phase 3: Allocate flexible tasks ("The Pour")
	for _, task := range flexibleTasks {
		slotsNeeded := durationToSlots(task.Duration)

		// Calculate deadline index if deadline exists
		var deadlineIdx int
		if task.Deadline != nil {
			deadlineIdx = timeToIndex(*task.Deadline)
			// If deadline is beyond window, treat as no constraint
			if deadlineIdx > totalSlots {
				deadlineIdx = totalSlots
			}
		} else {
			deadlineIdx = totalSlots
		}

		// Find first continuous gap that fits the task
		found := false
		for startIdx := 0; startIdx <= totalSlots-slotsNeeded; startIdx++ {
			// Check if this position would violate deadline
			endIdx := startIdx + slotsNeeded
			if endIdx > deadlineIdx {
				break // No point checking further, deadline constraint violated
			}

			// Check if all slots in this range are available
			allAvailable := true
			for i := startIdx; i < endIdx; i++ {
				if grid[i] {
					allAvailable = false
					// Optimization: skip to next available slot
					startIdx = i
					break
				}
			}

			if allAvailable {
				// Found a gap! Mark slots as occupied
				for i := startIdx; i < endIdx; i++ {
					grid[i] = true
				}

				// Convert indices back to slot-aligned times and add to result
				startTime := indexToTime(startIdx)
				endTime := indexToTime(startIdx + slotsNeeded)

				planned = append(planned, domain.PlannedTask{
					ID:    task.ID,
					Start: startTime,
					End:   endTime,
				})

				found = true
				break
			}
		}

		// If no gap found, add to unscheduled list
		if !found {
			unscheduled = append(unscheduled, task)
		}
	}

	return domain.Schedule{
		Planned:     planned,
		Unscheduled: unscheduled,
	}
}
