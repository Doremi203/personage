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
	if loc == nil {
		loc = time.UTC
	}

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

		// Round the slot-aligned end up when the task spills past its boundary slot.
		if endTime.Sub(indexToTime(endIdx)) > 0 {
			endIdx++
		}

		// Grid occupancy is confined to the window, so clamp only the marked range — never the
		// stored End. A wall whose start lies beyond the window must keep its real slot-aligned
		// finish; clamping End to the window end would push it before Start.
		for i := max(startIdx, 0); i < min(endIdx, totalSlots); i++ {
			grid[i] = true
		}

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
			// task.Date denotes a calendar day; read it in loc to recover that day. A stored DATE
			// comes back as UTC midnight and a freshly parsed value as loc midnight — both resolve
			// to the same civil day under .In(loc) for east-of-UTC zones like Europe/Moscow.
			day := task.Date.In(loc)
			dayStart := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, loc)
			dayEnd := time.Date(day.Year(), day.Month(), day.Day()+1, 0, 0, 0, 0, loc)
			// Date is best-effort: ignore it when its day has already elapsed, or when it would push
			// the task past an explicit deadline — in both cases schedule the task normally instead.
			if dayEnd.After(planningStart) && (task.Deadline == nil || !dayStart.After(*task.Deadline)) {
				lowIdx = max(lowIdx, timeToIndex(dayStart))
				highIdx = min(highIdx, timeToIndex(dayEnd))
			}
		}

		startIdx, ok := findFreeGap(grid, sleep, lowIdx, highIdx, slotsNeeded, true)
		if !ok && task.Deadline != nil {
			// Sleep is a soft preference; let an explicit deadline override it rather than leaving
			// the task unscheduled past a deadline the user cares about.
			startIdx, ok = findFreeGap(grid, sleep, lowIdx, highIdx, slotsNeeded, false)
		}
		if !ok {
			unscheduled = append(unscheduled, task)
			continue
		}

		for i := startIdx; i < startIdx+slotsNeeded; i++ {
			grid[i] = true
		}
		planned = append(planned, domain.PlannedTask{
			ID:    task.ID,
			Start: indexToTime(startIdx),
			End:   indexToTime(startIdx + slotsNeeded),
		})
	}

	return domain.Schedule{
		Planned:     planned,
		Unscheduled: unscheduled,
	}
}

// findFreeGap returns the first start index in [lowIdx, highIdx) where slotsNeeded contiguous
// slots are free. When honorSleep is true, sleep slots are treated as occupied.
func findFreeGap(grid, sleep []bool, lowIdx, highIdx, slotsNeeded int, honorSleep bool) (int, bool) {
	for startIdx := lowIdx; startIdx+slotsNeeded <= highIdx; startIdx++ {
		endIdx := startIdx + slotsNeeded

		blocked := false
		for i := startIdx; i < endIdx; i++ {
			if grid[i] || (honorSleep && sleep[i]) {
				blocked = true
				// Skip past the blocking slot — next iteration starts at i+1.
				startIdx = i
				break
			}
		}

		if !blocked {
			return startIdx, true
		}
	}

	return 0, false
}

func isSleepSlot(slotStart time.Time, loc *time.Location) bool {
	local := slotStart.In(loc)
	hour, minute := local.Hour(), local.Minute()
	return hour < 5 || (hour == 5 && minute < 30)
}
