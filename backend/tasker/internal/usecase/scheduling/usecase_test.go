package scheduling

import (
	"context"
	"testing"
	"time"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/Doremi203/personage/backend/libs/go/log"
	"github.com/Doremi203/personage/backend/tasker/internal/domain"
	mock_domain "github.com/Doremi203/personage/backend/tasker/internal/domain/mock"
	"go.uber.org/mock/gomock"
)

var fixedNow = time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)

func clock() time.Time { return fixedNow }

func TestSchedulePendingTasks_NoUsers(t *testing.T) {
	ctrl := gomock.NewController(t)

	repo := mock_domain.NewMockTaskRepo(ctrl)
	repo.EXPECT().GetUsersWithUnplannedTasks(gomock.Any()).Return(nil, nil)
	notificationSender := mock_domain.NewMockNotificationsService(ctrl)
	notificationSender.EXPECT().Send(gomock.Any(), gomock.Any()).Times(0)

	uc := NewUseCase(log.Stub{}, repo, notificationSender, 24*time.Hour, clock)

	err := uc.SchedulePendingTasks(t.Context())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestSchedulePendingTasks_SingleUserSingleTask(t *testing.T) {
	ctrl := gomock.NewController(t)

	userID := domain.UserID("user-1")
	tasks := []domain.Task{
		{
			ID:       "task-1",
			UserID:   userID,
			Duration: 30 * time.Minute,
			Priority: 5,
			Status:   domain.TaskStatusUnplanned,
		},
	}

	repo := mock_domain.NewMockTaskRepo(ctrl)
	repo.EXPECT().GetUsersWithUnplannedTasks(gomock.Any()).Return([]domain.UserID{userID}, nil)
	repo.EXPECT().GetTasksByStatus(gomock.Any(), userID, domain.TaskStatusUnplanned).Return(tasks, nil)
	repo.EXPECT().UpdateTaskSchedule(gomock.Any(), domain.TaskID("task-1"), gomock.Any(), gomock.Any(), domain.TaskStatusPlanned).Return(nil)

	notificationSender := mock_domain.NewMockNotificationsService(ctrl)
	notificationSender.EXPECT().Send(gomock.Any(), gomock.Any()).Return(nil)

	uc := NewUseCase(log.Stub{}, repo, notificationSender, 24*time.Hour, clock)

	err := uc.SchedulePendingTasks(t.Context())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestSchedulePendingTasks_MultipleUsers(t *testing.T) {
	ctrl := gomock.NewController(t)

	user1 := domain.UserID("user-1")
	user2 := domain.UserID("user-2")

	repo := mock_domain.NewMockTaskRepo(ctrl)
	repo.EXPECT().GetUsersWithUnplannedTasks(gomock.Any()).Return([]domain.UserID{user1, user2}, nil)

	repo.EXPECT().GetTasksByStatus(gomock.Any(), user1, domain.TaskStatusUnplanned).Return([]domain.Task{
		{ID: "task-1", UserID: user1, Duration: 20 * time.Minute, Priority: 5, Status: domain.TaskStatusUnplanned},
	}, nil)
	repo.EXPECT().UpdateTaskSchedule(gomock.Any(), domain.TaskID("task-1"), gomock.Any(), gomock.Any(), domain.TaskStatusPlanned).Return(nil)

	repo.EXPECT().GetTasksByStatus(gomock.Any(), user2, domain.TaskStatusUnplanned).Return([]domain.Task{
		{ID: "task-2", UserID: user2, Duration: 15 * time.Minute, Priority: 8, Status: domain.TaskStatusUnplanned},
		{ID: "task-3", UserID: user2, Duration: 25 * time.Minute, Priority: 3, Status: domain.TaskStatusUnplanned},
	}, nil)
	repo.EXPECT().UpdateTaskSchedule(gomock.Any(), domain.TaskID("task-2"), gomock.Any(), gomock.Any(), domain.TaskStatusPlanned).Return(nil)
	repo.EXPECT().UpdateTaskSchedule(gomock.Any(), domain.TaskID("task-3"), gomock.Any(), gomock.Any(), domain.TaskStatusPlanned).Return(nil)

	notificationSender := mock_domain.NewMockNotificationsService(ctrl)
	notificationSender.EXPECT().Send(gomock.Any(), gomock.Any()).Return(nil).Times(2)

	uc := NewUseCase(log.Stub{}, repo, notificationSender, 24*time.Hour, clock)

	err := uc.SchedulePendingTasks(t.Context())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestSchedulePendingTasks_GetUsersError(t *testing.T) {
	ctrl := gomock.NewController(t)

	repo := mock_domain.NewMockTaskRepo(ctrl)
	repo.EXPECT().GetUsersWithUnplannedTasks(gomock.Any()).Return(nil, errors.Errorf("db connection failed"))

	notificationSender := mock_domain.NewMockNotificationsService(ctrl)
	notificationSender.EXPECT().Send(gomock.Any(), gomock.Any()).Times(0)

	uc := NewUseCase(log.Stub{}, repo, notificationSender, 24*time.Hour, clock)

	err := uc.SchedulePendingTasks(t.Context())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestSchedulePendingTasks_GetTasksErrorContinuesWithNextUser(t *testing.T) {
	ctrl := gomock.NewController(t)

	user1 := domain.UserID("user-1")
	user2 := domain.UserID("user-2")

	repo := mock_domain.NewMockTaskRepo(ctrl)
	repo.EXPECT().GetUsersWithUnplannedTasks(gomock.Any()).Return([]domain.UserID{user1, user2}, nil)

	// user1 fails
	repo.EXPECT().GetTasksByStatus(gomock.Any(), user1, domain.TaskStatusUnplanned).Return(nil, errors.Errorf("query failed"))

	// user2 succeeds
	repo.EXPECT().GetTasksByStatus(gomock.Any(), user2, domain.TaskStatusUnplanned).Return([]domain.Task{
		{ID: "task-2", UserID: user2, Duration: 20 * time.Minute, Priority: 5, Status: domain.TaskStatusUnplanned},
	}, nil)
	repo.EXPECT().UpdateTaskSchedule(gomock.Any(), domain.TaskID("task-2"), gomock.Any(), gomock.Any(), domain.TaskStatusPlanned).Return(nil)

	notificationSender := mock_domain.NewMockNotificationsService(ctrl)
	notificationSender.EXPECT().Send(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	uc := NewUseCase(log.Stub{}, repo, notificationSender, 24*time.Hour, clock)

	// Should not return error — per-user errors are logged and skipped
	err := uc.SchedulePendingTasks(t.Context())
	if err != nil {
		t.Fatalf("expected no error (per-user failures are logged), got %v", err)
	}
}

func TestSchedulePendingTasks_TaskTooLargeForWindow(t *testing.T) {
	ctrl := gomock.NewController(t)

	userID := domain.UserID("user-1")

	repo := mock_domain.NewMockTaskRepo(ctrl)
	repo.EXPECT().GetUsersWithUnplannedTasks(gomock.Any()).Return([]domain.UserID{userID}, nil)
	repo.EXPECT().GetTasksByStatus(gomock.Any(), userID, domain.TaskStatusUnplanned).Return([]domain.Task{
		{ID: "big-task", UserID: userID, Duration: 25 * time.Hour, Priority: 5, Status: domain.TaskStatusUnplanned},
	}, nil)
	// No UpdateTaskSchedule call expected — task doesn't fit

	notificationSender := mock_domain.NewMockNotificationsService(ctrl)

	uc := NewUseCase(log.Stub{}, repo, notificationSender, 24*time.Hour, clock)

	err := uc.SchedulePendingTasks(t.Context())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestSchedulePendingTasks_PriorityOrdering(t *testing.T) {
	ctrl := gomock.NewController(t)

	userID := domain.UserID("user-1")

	repo := mock_domain.NewMockTaskRepo(ctrl)
	repo.EXPECT().GetUsersWithUnplannedTasks(gomock.Any()).Return([]domain.UserID{userID}, nil)
	repo.EXPECT().GetTasksByStatus(gomock.Any(), userID, domain.TaskStatusUnplanned).Return([]domain.Task{
		{ID: "low-priority", UserID: userID, Duration: 20 * time.Minute, Priority: 1, Status: domain.TaskStatusUnplanned},
		{ID: "high-priority", UserID: userID, Duration: 20 * time.Minute, Priority: 10, Status: domain.TaskStatusUnplanned},
	}, nil)

	var highStart, lowStart time.Time
	repo.EXPECT().UpdateTaskSchedule(gomock.Any(), domain.TaskID("high-priority"), gomock.Any(), gomock.Any(), domain.TaskStatusPlanned).
		DoAndReturn(func(_ context.Context, _ domain.TaskID, start time.Time, _ time.Time, _ domain.TaskStatus) error {
			highStart = start
			return nil
		})
	repo.EXPECT().UpdateTaskSchedule(gomock.Any(), domain.TaskID("low-priority"), gomock.Any(), gomock.Any(), domain.TaskStatusPlanned).
		DoAndReturn(func(_ context.Context, _ domain.TaskID, start time.Time, _ time.Time, _ domain.TaskStatus) error {
			lowStart = start
			return nil
		})

	notificationSender := mock_domain.NewMockNotificationsService(ctrl)
	notificationSender.EXPECT().Send(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	uc := NewUseCase(log.Stub{}, repo, notificationSender, 1*time.Hour, clock)

	err := uc.SchedulePendingTasks(t.Context())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !highStart.Before(lowStart) {
		t.Errorf("high priority task should start before low priority: high=%v, low=%v", highStart, lowStart)
	}
}

func TestSchedulePendingTasks_UpdateErrorFailsUserContinuesNext(t *testing.T) {
	ctrl := gomock.NewController(t)

	user1 := domain.UserID("user-1")
	user2 := domain.UserID("user-2")

	repo := mock_domain.NewMockTaskRepo(ctrl)
	repo.EXPECT().GetUsersWithUnplannedTasks(gomock.Any()).Return([]domain.UserID{user1, user2}, nil)

	// user1: task update fails
	repo.EXPECT().GetTasksByStatus(gomock.Any(), user1, domain.TaskStatusUnplanned).Return([]domain.Task{
		{ID: "task-1", UserID: user1, Duration: 20 * time.Minute, Priority: 5, Status: domain.TaskStatusUnplanned},
	}, nil)
	repo.EXPECT().UpdateTaskSchedule(gomock.Any(), domain.TaskID("task-1"), gomock.Any(), gomock.Any(), domain.TaskStatusPlanned).
		Return(errors.Errorf("simulated update failure"))

	// user2: succeeds
	repo.EXPECT().GetTasksByStatus(gomock.Any(), user2, domain.TaskStatusUnplanned).Return([]domain.Task{
		{ID: "task-2", UserID: user2, Duration: 20 * time.Minute, Priority: 5, Status: domain.TaskStatusUnplanned},
	}, nil)
	repo.EXPECT().UpdateTaskSchedule(gomock.Any(), domain.TaskID("task-2"), gomock.Any(), gomock.Any(), domain.TaskStatusPlanned).
		Return(nil)

	notificationSender := mock_domain.NewMockNotificationsService(ctrl)
	notificationSender.EXPECT().Send(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	uc := NewUseCase(log.Stub{}, repo, notificationSender, 24*time.Hour, clock)

	// Should not return error — per-user failures are logged and skipped
	err := uc.SchedulePendingTasks(t.Context())
	if err != nil {
		t.Fatalf("expected no error (per-user failures are logged), got %v", err)
	}
}
