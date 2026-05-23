package scheduler

import (
	"math"
	"sort"
	"time"

	"github.com/Doremi203/personage/backend/tasker/internal/domain"
)

func CalculateSchedule(
	tasks []domain.Task,
	planningStart time.Time,
	windowDuration time.Duration,
	loc *time.Location,
) domain.Schedule {
	totalSlots := int(math.Ceil(float64(windowDuration) / float64(domain.TimeSlotSize)))

	grid := make([]bool, totalSlots)

	var planned []domain.PlannedTask
	var unscheduled []domain.Task

	timeToIndex := func(t time.Time) int {
		diff := t.Sub(planningStart)
		return int(diff / domain.TimeSlotSize)
	}

	indexToTime := func(idx int) time.Time {
		return planningStart.Add(time.Duration(idx) * domain.TimeSlotSize)
	}

	durationToSlots := func(d time.Duration) int {
		return int(math.Ceil(float64(d) / float64(domain.TimeSlotSize)))
	}

	sleep := make([]bool, totalSlots)
	for i := range totalSlots {
		if isSleepSlot(indexToTime(i), loc) {
			sleep[i] = true
		}
	}

	// Phase 1: Process fixed tasks ("The Walls").
	// Explicit start_time is respected as-is — sleep window is NOT applied here.
	for _, task := range tasks {
		if task.StartTime == nil {
			continue
		}

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

	// Phase 3: Allocate flexible tasks ("The Pour").
	// Honors deadline, optional best-effort date (day box), and the 00:00–05:30 Moscow sleep window.
	for _, task := range flexibleTasks {
		slotsNeeded := durationToSlots(task.Duration)

		lowIdx := 0
		highIdx := totalSlots
		if task.Deadline != nil {
			highIdx = min(highIdx, timeToIndex(*task.Deadline))
		}
		if task.Date != nil {
			mskDate := task.Date.In(loc)
			dayStart := time.Date(mskDate.Year(), mskDate.Month(), mskDate.Day(), 0, 0, 0, 0, loc).UTC()
			dayEnd := dayStart.Add(24 * time.Hour)
			// Date is best-effort: ignore it when it pushes the task past an explicit deadline.
			if task.Deadline == nil || !dayStart.After(*task.Deadline) {
				lowIdx = max(lowIdx, timeToIndex(dayStart))
				highIdx = min(highIdx, timeToIndex(dayEnd))
			}
		}

		found := false
		for startIdx := lowIdx; startIdx+slotsNeeded <= highIdx; startIdx++ {
			endIdx := startIdx + slotsNeeded

			allAvailable := true
			for i := startIdx; i < endIdx; i++ {
				if grid[i] || sleep[i] {
					allAvailable = false
					// Skip past the blocking slot — next iteration starts at i+1.
					startIdx = i
					break
				}
			}

			if allAvailable {
				for i := startIdx; i < endIdx; i++ {
					grid[i] = true
				}

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

		if !found {
			unscheduled = append(unscheduled, task)
		}
	}

	return domain.Schedule{
		Planned:     planned,
		Unscheduled: unscheduled,
	}
}

func isSleepSlot(slotStart time.Time, loc *time.Location) bool {
	local := slotStart.In(loc)
	hour, minute := local.Hour(), local.Minute()
	return hour < 5 || (hour == 5 && minute < 30)
}
