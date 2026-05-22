package notifications_test

import (
	"testing"

	"github.com/Doremi203/personage/backend/libs/go/log"
	"github.com/Doremi203/personage/backend/tasker/internal/domain"
	mock_domain "github.com/Doremi203/personage/backend/tasker/internal/domain/mock"
	"github.com/Doremi203/personage/backend/tasker/internal/services/notifications"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestWorker_Process(t *testing.T) {
	type mocks struct {
		taskRepo              *mock_domain.MockTaskRepo
		upcomingEventNotifier *mock_domain.MockUpcomingEventNotifier
	}
	type args struct {
		userIDs      []domain.UserID
		plannedTasks map[domain.UserID][]domain.Task
	}

	user1 := domain.UserID("user-1")
	user2 := domain.UserID("user-2")
	task1 := domain.Task{ID: "task-1", UserID: user1, Title: "Test", Status: domain.TaskStatusPlanned}
	task2 := domain.Task{ID: "task-2", UserID: user2, Title: "Test 2", Status: domain.TaskStatusPlanned}

	tests := []struct {
		name    string
		args    args
		setup   func(m mocks, a args)
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "GetUsersWithPlannedTasks error wrapped and returned",
			setup: func(m mocks, _ args) {
				m.taskRepo.EXPECT().
					GetUsersWithPlannedTasks(gomock.Any()).
					Return(nil, assert.AnError)
			},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.ErrorIs(t, err, assert.AnError)
			},
		},
		{
			name: "empty users list returns nil with no further calls",
			setup: func(m mocks, _ args) {
				m.taskRepo.EXPECT().
					GetUsersWithPlannedTasks(gomock.Any()).
					Return(nil, nil)
			},
			wantErr: require.NoError,
		},
		{
			name: "single user with empty planned tasks does not call notifier",
			args: args{
				userIDs: []domain.UserID{user1},
			},
			setup: func(m mocks, a args) {
				m.taskRepo.EXPECT().
					GetUsersWithPlannedTasks(gomock.Any()).
					Return(a.userIDs, nil)
				m.taskRepo.EXPECT().
					GetTasksByStatus(gomock.Any(), user1, domain.TaskStatusPlanned).
					Return(nil, nil)
			},
			wantErr: require.NoError,
		},
		{
			name: "single user with planned tasks calls notifier",
			args: args{
				userIDs:      []domain.UserID{user1},
				plannedTasks: map[domain.UserID][]domain.Task{user1: {task1}},
			},
			setup: func(m mocks, a args) {
				m.taskRepo.EXPECT().
					GetUsersWithPlannedTasks(gomock.Any()).
					Return(a.userIDs, nil)
				m.taskRepo.EXPECT().
					GetTasksByStatus(gomock.Any(), user1, domain.TaskStatusPlanned).
					Return(a.plannedTasks[user1], nil)
				m.upcomingEventNotifier.EXPECT().
					NotifyUpcomingEvents(gomock.Any(), user1, a.plannedTasks[user1]).
					Return(nil)
			},
			wantErr: require.NoError,
		},
		{
			name: "single user GetTasksByStatus error is logged and loop continues",
			args: args{
				userIDs: []domain.UserID{user1},
			},
			setup: func(m mocks, a args) {
				m.taskRepo.EXPECT().
					GetUsersWithPlannedTasks(gomock.Any()).
					Return(a.userIDs, nil)
				m.taskRepo.EXPECT().
					GetTasksByStatus(gomock.Any(), user1, domain.TaskStatusPlanned).
					Return(nil, assert.AnError)
			},
			wantErr: require.NoError,
		},
		{
			name: "single user notifier error is logged and loop continues",
			args: args{
				userIDs:      []domain.UserID{user1},
				plannedTasks: map[domain.UserID][]domain.Task{user1: {task1}},
			},
			setup: func(m mocks, a args) {
				m.taskRepo.EXPECT().
					GetUsersWithPlannedTasks(gomock.Any()).
					Return(a.userIDs, nil)
				m.taskRepo.EXPECT().
					GetTasksByStatus(gomock.Any(), user1, domain.TaskStatusPlanned).
					Return(a.plannedTasks[user1], nil)
				m.upcomingEventNotifier.EXPECT().
					NotifyUpcomingEvents(gomock.Any(), user1, a.plannedTasks[user1]).
					Return(assert.AnError)
			},
			wantErr: require.NoError,
		},
		{
			name: "multiple users one fails GetTasksByStatus another succeeds",
			args: args{
				userIDs:      []domain.UserID{user1, user2},
				plannedTasks: map[domain.UserID][]domain.Task{user2: {task2}},
			},
			setup: func(m mocks, a args) {
				m.taskRepo.EXPECT().
					GetUsersWithPlannedTasks(gomock.Any()).
					Return(a.userIDs, nil)
				m.taskRepo.EXPECT().
					GetTasksByStatus(gomock.Any(), user1, domain.TaskStatusPlanned).
					Return(nil, assert.AnError)
				m.taskRepo.EXPECT().
					GetTasksByStatus(gomock.Any(), user2, domain.TaskStatusPlanned).
					Return(a.plannedTasks[user2], nil)
				m.upcomingEventNotifier.EXPECT().
					NotifyUpcomingEvents(gomock.Any(), user2, a.plannedTasks[user2]).
					Return(nil)
			},
			wantErr: require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			m := mocks{
				taskRepo:              mock_domain.NewMockTaskRepo(ctrl),
				upcomingEventNotifier: mock_domain.NewMockUpcomingEventNotifier(ctrl),
			}
			tt.setup(m, tt.args)

			worker := notifications.NewWorker(
				log.Stub{},
				m.taskRepo,
				m.upcomingEventNotifier,
			)

			err := worker.Process(t.Context())
			tt.wantErr(t, err)
		})
	}
}
