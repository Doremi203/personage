package notifications

import (
	"testing"
	"time"

	"github.com/Doremi203/personage/backend/tasker/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

func TestFormatUpcomingEventTitle(t *testing.T) {
	task := domain.Task{
		ID:       "task-1",
		Title:    "Test Task",
		Priority: 5,
		Status:   domain.TaskStatusPlanned,
		Duration: 30 * time.Minute,
		Category: domain.TaskCategoryWork,
	}

	tests := []struct {
		name     string
		interval time.Duration
		want     string
	}{
		{
			name:     "1 hour",
			interval: 1 * time.Hour,
			want:     "⏰ [Test Task] через 1 час",
		},
		{
			name:     "2 hours",
			interval: 2 * time.Hour,
			want:     "⏰ [Test Task] через 2 часа",
		},
		{
			name:     "5 hours",
			interval: 5 * time.Hour,
			want:     "⏰ [Test Task] через 5 часов",
		},
		{
			name:     "1 minute",
			interval: 1 * time.Minute,
			want:     "⏰ [Test Task] через 1 минуту",
		},
		{
			name:     "2 minutes",
			interval: 2 * time.Minute,
			want:     "⏰ [Test Task] через 2 минуты",
		},
		{
			name:     "5 minutes",
			interval: 5 * time.Minute,
			want:     "⏰ [Test Task] через 5 минут",
		},
		{
			name:     "15 minutes",
			interval: 15 * time.Minute,
			want:     "⏰ [Test Task] через 15 минут",
		},
		{
			name:     "21 minutes",
			interval: 21 * time.Minute,
			want:     "⏰ [Test Task] через 21 минуту",
		},
		{
			name:     "22 minutes",
			interval: 22 * time.Minute,
			want:     "⏰ [Test Task] через 22 минуты",
		},
		{
			name:     "25 minutes",
			interval: 25 * time.Minute,
			want:     "⏰ [Test Task] через 25 минут",
		},
		{
			name:     "30 minutes",
			interval: 30 * time.Minute,
			want:     "⏰ [Test Task] через 30 минут",
		},
		{
			name:     "31 minutes",
			interval: 31 * time.Minute,
			want:     "⏰ [Test Task] через 31 минуту",
		},
		{
			name:     "32 minutes",
			interval: 32 * time.Minute,
			want:     "⏰ [Test Task] через 32 минуты",
		},
		{
			name:     "35 minutes",
			interval: 35 * time.Minute,
			want:     "⏰ [Test Task] через 35 минут",
		},
		{
			name:     "40 minutes",
			interval: 40 * time.Minute,
			want:     "⏰ [Test Task] через 40 минут",
		},
		{
			name:     "41 minutes",
			interval: 41 * time.Minute,
			want:     "⏰ [Test Task] через 41 минуту",
		},
		{
			name:     "42 minutes",
			interval: 42 * time.Minute,
			want:     "⏰ [Test Task] через 42 минуты",
		},
		{
			name:     "45 minutes",
			interval: 45 * time.Minute,
			want:     "⏰ [Test Task] через 45 минут",
		},
		{
			name:     "50 minutes",
			interval: 50 * time.Minute,
			want:     "⏰ [Test Task] через 50 минут",
		},
		{
			name:     "51 minutes",
			interval: 51 * time.Minute,
			want:     "⏰ [Test Task] через 51 минуту",
		},
		{
			name:     "52 minutes",
			interval: 52 * time.Minute,
			want:     "⏰ [Test Task] через 52 минуты",
		},
		{
			name:     "55 minutes",
			interval: 55 * time.Minute,
			want:     "⏰ [Test Task] через 55 минут",
		},
		{
			name:     "60 minutes",
			interval: 60 * time.Minute,
			want:     "⏰ [Test Task] через 1 час",
		},
		{
			name:     "15 seconds",
			interval: 15 * time.Second,
			want:     "⏰ [Test Task] скоро начнется",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notifier, err := NewUpcomingEventNotifier(
				nil,
				domain.NotificationConfig{},
				message.NewPrinter(language.Russian),
			)
			require.NoError(t, err)

			got := notifier.formatUpcomingEventTitle(task, tt.interval)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFormatUpcomingEventBody(t *testing.T) {
	startTime := time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC)

	tests := []struct {
		name string
		task domain.Task
		want string
	}{
		{
			name: "30 minutes duration",
			task: domain.Task{
				ID:        "task-1",
				Title:     "Test Task",
				Priority:  5,
				Status:    domain.TaskStatusPlanned,
				Duration:  30 * time.Minute,
				Category:  domain.TaskCategoryWork,
				StartTime: &startTime,
			},
			want: "Начало в " + startTime.Format("15:04") + " • Длительность: 30 минут • Категория: Работа",
		},
		{
			name: "1 minute duration",
			task: domain.Task{
				ID:        "task-1",
				Title:     "Test Task",
				Priority:  5,
				Status:    domain.TaskStatusPlanned,
				Duration:  1 * time.Minute,
				Category:  domain.TaskCategoryWork,
				StartTime: &startTime,
			},
			want: "Начало в " + startTime.Format("15:04") + " • Длительность: 1 минута • Категория: Работа",
		},
		{
			name: "2 minutes duration",
			task: domain.Task{
				ID:        "task-1",
				Title:     "Test Task",
				Priority:  5,
				Status:    domain.TaskStatusPlanned,
				Duration:  2 * time.Minute,
				Category:  domain.TaskCategoryWork,
				StartTime: &startTime,
			},
			want: "Начало в " + startTime.Format("15:04") + " • Длительность: 2 минуты • Категория: Работа",
		},
		{
			name: "5 minutes duration",
			task: domain.Task{
				ID:        "task-1",
				Title:     "Test Task",
				Priority:  5,
				Status:    domain.TaskStatusPlanned,
				Duration:  5 * time.Minute,
				Category:  domain.TaskCategoryWork,
				StartTime: &startTime,
			},
			want: "Начало в " + startTime.Format("15:04") + " • Длительность: 5 минут • Категория: Работа",
		},
		{
			name: "21 minutes duration",
			task: domain.Task{
				ID:        "task-1",
				Title:     "Test Task",
				Priority:  5,
				Status:    domain.TaskStatusPlanned,
				Duration:  21 * time.Minute,
				Category:  domain.TaskCategoryWork,
				StartTime: &startTime,
			},
			want: "Начало в " + startTime.Format("15:04") + " • Длительность: 21 минута • Категория: Работа",
		},
		{
			name: "22 minutes duration",
			task: domain.Task{
				ID:        "task-1",
				Title:     "Test Task",
				Priority:  5,
				Status:    domain.TaskStatusPlanned,
				Duration:  22 * time.Minute,
				Category:  domain.TaskCategoryWork,
				StartTime: &startTime,
			},
			want: "Начало в " + startTime.Format("15:04") + " • Длительность: 22 минуты • Категория: Работа",
		},
		{
			name: "25 minutes duration",
			task: domain.Task{
				ID:        "task-1",
				Title:     "Test Task",
				Priority:  5,
				Status:    domain.TaskStatusPlanned,
				Duration:  25 * time.Minute,
				Category:  domain.TaskCategoryWork,
				StartTime: &startTime,
			},
			want: "Начало в " + startTime.Format("15:04") + " • Длительность: 25 минут • Категория: Работа",
		},
		{
			name: "31 minutes duration",
			task: domain.Task{
				ID:        "task-1",
				Title:     "Test Task",
				Priority:  5,
				Status:    domain.TaskStatusPlanned,
				Duration:  31 * time.Minute,
				Category:  domain.TaskCategoryWork,
				StartTime: &startTime,
			},
			want: "Начало в " + startTime.Format("15:04") + " • Длительность: 31 минута • Категория: Работа",
		},
		{
			name: "32 minutes duration",
			task: domain.Task{
				ID:        "task-1",
				Title:     "Test Task",
				Priority:  5,
				Status:    domain.TaskStatusPlanned,
				Duration:  32 * time.Minute,
				Category:  domain.TaskCategoryWork,
				StartTime: &startTime,
			},
			want: "Начало в " + startTime.Format("15:04") + " • Длительность: 32 минуты • Категория: Работа",
		},
		{
			name: "35 minutes duration",
			task: domain.Task{
				ID:        "task-1",
				Title:     "Test Task",
				Priority:  5,
				Status:    domain.TaskStatusPlanned,
				Duration:  35 * time.Minute,
				Category:  domain.TaskCategoryWork,
				StartTime: &startTime,
			},
			want: "Начало в " + startTime.Format("15:04") + " • Длительность: 35 минут • Категория: Работа",
		},
		{
			name: "41 minutes duration",
			task: domain.Task{
				ID:        "task-1",
				Title:     "Test Task",
				Priority:  5,
				Status:    domain.TaskStatusPlanned,
				Duration:  41 * time.Minute,
				Category:  domain.TaskCategoryWork,
				StartTime: &startTime,
			},
			want: "Начало в " + startTime.Format("15:04") + " • Длительность: 41 минута • Категория: Работа",
		},
		{
			name: "42 minutes duration",
			task: domain.Task{
				ID:        "task-1",
				Title:     "Test Task",
				Priority:  5,
				Status:    domain.TaskStatusPlanned,
				Duration:  42 * time.Minute,
				Category:  domain.TaskCategoryWork,
				StartTime: &startTime,
			},
			want: "Начало в " + startTime.Format("15:04") + " • Длительность: 42 минуты • Категория: Работа",
		},
		{
			name: "45 minutes duration",
			task: domain.Task{
				ID:        "task-1",
				Title:     "Test Task",
				Priority:  5,
				Status:    domain.TaskStatusPlanned,
				Duration:  45 * time.Minute,
				Category:  domain.TaskCategoryWork,
				StartTime: &startTime,
			},
			want: "Начало в " + startTime.Format("15:04") + " • Длительность: 45 минут • Категория: Работа",
		},
		{
			name: "51 minutes duration",
			task: domain.Task{
				ID:        "task-1",
				Title:     "Test Task",
				Priority:  5,
				Status:    domain.TaskStatusPlanned,
				Duration:  51 * time.Minute,
				Category:  domain.TaskCategoryWork,
				StartTime: &startTime,
			},
			want: "Начало в " + startTime.Format("15:04") + " • Длительность: 51 минута • Категория: Работа",
		},
		{
			name: "52 minutes duration",
			task: domain.Task{
				ID:        "task-1",
				Title:     "Test Task",
				Priority:  5,
				Status:    domain.TaskStatusPlanned,
				Duration:  52 * time.Minute,
				Category:  domain.TaskCategoryWork,
				StartTime: &startTime,
			},
			want: "Начало в " + startTime.Format("15:04") + " • Длительность: 52 минуты • Категория: Работа",
		},
		{
			name: "55 minutes duration",
			task: domain.Task{
				ID:        "task-1",
				Title:     "Test Task",
				Priority:  5,
				Status:    domain.TaskStatusPlanned,
				Duration:  55 * time.Minute,
				Category:  domain.TaskCategoryWork,
				StartTime: &startTime,
			},
			want: "Начало в " + startTime.Format("15:04") + " • Длительность: 55 минут • Категория: Работа",
		},
		{
			name: "60 minutes duration",
			task: domain.Task{
				ID:        "task-1",
				Title:     "Test Task",
				Priority:  5,
				Status:    domain.TaskStatusPlanned,
				Duration:  60 * time.Minute,
				Category:  domain.TaskCategoryWork,
				StartTime: &startTime,
			},
			want: "Начало в " + startTime.Format("15:04") + " • Длительность: 60 минут • Категория: Работа",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notifier, err := NewUpcomingEventNotifier(
				nil,
				domain.NotificationConfig{},
				message.NewPrinter(language.Russian),
			)
			require.NoError(t, err)
			got := notifier.formatUpcomingEventBody(tt.task)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFormatScheduleChangeBody(t *testing.T) {
	notifier := NewScheduleChangeNotifier(nil, domain.NotificationConfig{})

	oldStart := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	oldEnd := time.Date(2024, 1, 15, 11, 0, 0, 0, time.UTC)
	newStart := time.Date(2024, 1, 15, 14, 0, 0, 0, time.UTC)
	newEnd := time.Date(2024, 1, 15, 15, 30, 0, 0, time.UTC)

	tests := []struct {
		name   string
		change domain.ScheduleChange
		want   string
	}{
		{
			name: "new task with time",
			change: domain.ScheduleChange{
				TaskID:     "task-1",
				TaskTitle:  "Новая задача",
				NewStart:   &newStart,
				NewEnd:     &newEnd,
				ChangeType: domain.ScheduleChangeTypeNew,
			},
			want: "Задача: Новая задача\nВремя: 14:00 - 15:30",
		},
		{
			name: "new task without time",
			change: domain.ScheduleChange{
				TaskID:     "task-1",
				TaskTitle:  "Новая задача",
				ChangeType: domain.ScheduleChangeTypeNew,
			},
			want: "Задача: Новая задача\n",
		},
		{
			name: "modified task with both old and new times",
			change: domain.ScheduleChange{
				TaskID:     "task-1",
				TaskTitle:  "Измененная задача",
				OldStart:   &oldStart,
				OldEnd:     &oldEnd,
				NewStart:   &newStart,
				NewEnd:     &newEnd,
				ChangeType: domain.ScheduleChangeTypeModified,
			},
			want: "Задача: Измененная задача\nИзменено с 10:00-11:00 на 14:00-15:30",
		},
		{
			name: "modified task with only old time (removed from schedule)",
			change: domain.ScheduleChange{
				TaskID:     "task-1",
				TaskTitle:  "Удаленная из расписания",
				OldStart:   &oldStart,
				OldEnd:     &oldEnd,
				ChangeType: domain.ScheduleChangeTypeModified,
			},
			want: "Задача: Удаленная из расписания\nУдалено из 10:00-11:00",
		},
		{
			name: "modified task with only new time (added to schedule)",
			change: domain.ScheduleChange{
				TaskID:     "task-1",
				TaskTitle:  "Добавленная в расписание",
				NewStart:   &newStart,
				NewEnd:     &newEnd,
				ChangeType: domain.ScheduleChangeTypeModified,
			},
			want: "Задача: Добавленная в расписание\nЗапланировано на 14:00-15:30",
		},
		{
			name: "modified task without any times",
			change: domain.ScheduleChange{
				TaskID:     "task-1",
				TaskTitle:  "Измененная задача",
				ChangeType: domain.ScheduleChangeTypeModified,
			},
			want: "Задача: Измененная задача\n",
		},
		{
			name: "deleted task with old time",
			change: domain.ScheduleChange{
				TaskID:     "task-1",
				TaskTitle:  "Удаленная задача",
				OldStart:   &oldStart,
				OldEnd:     &oldEnd,
				ChangeType: domain.ScheduleChangeTypeDeleted,
			},
			want: "Задача: Удаленная задача\nБыло запланировано: 10:00 - 11:00",
		},
		{
			name: "deleted task without old time",
			change: domain.ScheduleChange{
				TaskID:     "task-1",
				TaskTitle:  "Удаленная задача",
				ChangeType: domain.ScheduleChangeTypeDeleted,
			},
			want: "Задача: Удаленная задача\n",
		},
		{
			name: "unknown change type",
			change: domain.ScheduleChange{
				TaskID:     "task-1",
				TaskTitle:  "Неизвестное изменение",
				ChangeType: "unknown",
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := notifier.formatScheduleChangeBody(tt.change)
			assert.Equal(t, tt.want, got)
		})
	}
}
