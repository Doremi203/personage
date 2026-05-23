package tasklist_test

import (
	"context"
	"testing"
	"time"

	"github.com/Doremi203/personage/backend/libs/go/tx"
	"github.com/Doremi203/personage/backend/tasker/internal/domain"
	mock_domain "github.com/Doremi203/personage/backend/tasker/internal/domain/mock"
	"github.com/Doremi203/personage/backend/tasker/internal/usecase/tasklist"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const (
	testTaskID = "task-1"
	testUserID = "user-1"
)

func TestUseCase_GetTask(t *testing.T) {
	type mocks struct {
		taskRepo    *mock_domain.MockTaskRepo
		eventRepo   *mock_domain.MockEventRepo
		clusterRepo *mock_domain.MockClusterRepo
	}
	type args struct {
		taskID string
		userID string
	}

	wantTask := domain.Task{ID: domain.TaskID(testTaskID), UserID: domain.UserID(testUserID), Title: "title"}

	tests := []struct {
		name    string
		args    args
		setup   func(m mocks, a args)
		want    domain.Task
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "success",
			args: args{taskID: testTaskID, userID: testUserID},
			setup: func(m mocks, a args) {
				m.taskRepo.EXPECT().
					GetTaskByID(gomock.Any(), domain.TaskID(a.taskID), domain.UserID(a.userID)).
					Return(wantTask, nil)
			},
			want:    wantTask,
			wantErr: require.NoError,
		},
		{
			name: "repo error wraps",
			args: args{taskID: testTaskID, userID: testUserID},
			setup: func(m mocks, a args) {
				m.taskRepo.EXPECT().
					GetTaskByID(gomock.Any(), domain.TaskID(a.taskID), domain.UserID(a.userID)).
					Return(domain.Task{}, assert.AnError)
			},
			want:    domain.Task{},
			wantErr: require.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			m := mocks{
				taskRepo:    mock_domain.NewMockTaskRepo(ctrl),
				eventRepo:   mock_domain.NewMockEventRepo(ctrl),
				clusterRepo: mock_domain.NewMockClusterRepo(ctrl),
			}
			tt.setup(m, tt.args)

			uc := tasklist.NewUseCase(m.taskRepo, m.eventRepo, m.clusterRepo, stubTxProvider{})
			got, err := uc.GetTask(t.Context(), tt.args.taskID, tt.args.userID)

			tt.wantErr(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestUseCase_UpdateTask(t *testing.T) {
	type mocks struct {
		taskRepo    *mock_domain.MockTaskRepo
		eventRepo   *mock_domain.MockEventRepo
		clusterRepo *mock_domain.MockClusterRepo
	}
	type args struct {
		taskID string
		userID string
		update domain.TaskUpdate
	}

	newTitle := "new title"
	updatedTask := domain.Task{ID: domain.TaskID(testTaskID), Title: newTitle}
	startTime := time.Date(2026, 5, 23, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		args    args
		setup   func(m mocks, a args)
		want    domain.Task
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "success",
			args: args{taskID: testTaskID, userID: testUserID, update: domain.TaskUpdate{Title: &newTitle}},
			setup: func(m mocks, a args) {
				m.taskRepo.EXPECT().
					UpdateTask(gomock.Any(), domain.TaskID(a.taskID), domain.UserID(a.userID), domain.TaskUpdate{Title: &newTitle}).
					Return(updatedTask, nil)
			},
			want:    updatedTask,
			wantErr: require.NoError,
		},
		{
			name: "setting start time auto-plans task",
			args: args{taskID: testTaskID, userID: testUserID, update: domain.TaskUpdate{StartTime: &startTime}},
			setup: func(m mocks, a args) {
				m.taskRepo.EXPECT().
					UpdateTask(gomock.Any(), domain.TaskID(a.taskID), domain.UserID(a.userID), domain.TaskUpdate{
						StartTime: &startTime,
						Status:    new(domain.TaskStatusPlanned),
					}).
					Return(updatedTask, nil)
			},
			want:    updatedTask,
			wantErr: require.NoError,
		},
		{
			name: "explicit status overrides auto-plan",
			args: args{taskID: testTaskID, userID: testUserID, update: domain.TaskUpdate{
				StartTime: &startTime,
				Status:    new(domain.TaskStatusCompleted),
			}},
			setup: func(m mocks, a args) {
				m.taskRepo.EXPECT().
					UpdateTask(gomock.Any(), domain.TaskID(a.taskID), domain.UserID(a.userID), domain.TaskUpdate{
						StartTime: &startTime,
						Status:    new(domain.TaskStatusCompleted),
					}).
					Return(updatedTask, nil)
			},
			want:    updatedTask,
			wantErr: require.NoError,
		},
		{
			name: "repo error wraps",
			args: args{taskID: testTaskID, userID: testUserID, update: domain.TaskUpdate{Title: &newTitle}},
			setup: func(m mocks, a args) {
				m.taskRepo.EXPECT().
					UpdateTask(gomock.Any(), domain.TaskID(a.taskID), domain.UserID(a.userID), domain.TaskUpdate{Title: &newTitle}).
					Return(domain.Task{}, assert.AnError)
			},
			want:    domain.Task{},
			wantErr: require.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			m := mocks{
				taskRepo:    mock_domain.NewMockTaskRepo(ctrl),
				eventRepo:   mock_domain.NewMockEventRepo(ctrl),
				clusterRepo: mock_domain.NewMockClusterRepo(ctrl),
			}
			tt.setup(m, tt.args)

			uc := tasklist.NewUseCase(m.taskRepo, m.eventRepo, m.clusterRepo, stubTxProvider{})
			got, err := uc.UpdateTask(t.Context(), tt.args.taskID, tt.args.userID, tt.args.update)

			tt.wantErr(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestUseCase_PostponeTask(t *testing.T) {
	type mocks struct {
		taskRepo    *mock_domain.MockTaskRepo
		eventRepo   *mock_domain.MockEventRepo
		clusterRepo *mock_domain.MockClusterRepo
	}
	type args struct {
		taskID string
		userID string
	}

	existing := domain.Task{
		ID:     domain.TaskID(testTaskID),
		UserID: domain.UserID(testUserID),
		Status: domain.TaskStatusPlanned,
		Title:  "title",
	}
	want := existing
	want.Status = domain.TaskStatusUnplanned

	tests := []struct {
		name    string
		args    args
		setup   func(m mocks, a args)
		want    domain.Task
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "success",
			args: args{taskID: testTaskID, userID: testUserID},
			setup: func(m mocks, a args) {
				m.taskRepo.EXPECT().
					GetTaskByID(gomock.Any(), domain.TaskID(a.taskID), domain.UserID(a.userID)).
					Return(existing, nil)
				m.taskRepo.EXPECT().
					UpdateTaskStatus(gomock.Any(), domain.TaskID(a.taskID), domain.TaskStatusUnplanned).
					Return(nil)
			},
			want:    want,
			wantErr: require.NoError,
		},
		{
			name: "GetTaskByID error wraps",
			args: args{taskID: testTaskID, userID: testUserID},
			setup: func(m mocks, a args) {
				m.taskRepo.EXPECT().
					GetTaskByID(gomock.Any(), domain.TaskID(a.taskID), domain.UserID(a.userID)).
					Return(domain.Task{}, assert.AnError)
			},
			want:    domain.Task{},
			wantErr: require.Error,
		},
		{
			name: "UpdateTaskStatus error wraps",
			args: args{taskID: testTaskID, userID: testUserID},
			setup: func(m mocks, a args) {
				m.taskRepo.EXPECT().
					GetTaskByID(gomock.Any(), domain.TaskID(a.taskID), domain.UserID(a.userID)).
					Return(existing, nil)
				m.taskRepo.EXPECT().
					UpdateTaskStatus(gomock.Any(), domain.TaskID(a.taskID), domain.TaskStatusUnplanned).
					Return(assert.AnError)
			},
			want:    domain.Task{},
			wantErr: require.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			m := mocks{
				taskRepo:    mock_domain.NewMockTaskRepo(ctrl),
				eventRepo:   mock_domain.NewMockEventRepo(ctrl),
				clusterRepo: mock_domain.NewMockClusterRepo(ctrl),
			}
			tt.setup(m, tt.args)

			uc := tasklist.NewUseCase(m.taskRepo, m.eventRepo, m.clusterRepo, stubTxProvider{})
			got, err := uc.PostponeTask(t.Context(), tt.args.taskID, tt.args.userID)

			tt.wantErr(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestUseCase_CompleteTask(t *testing.T) {
	type mocks struct {
		taskRepo    *mock_domain.MockTaskRepo
		eventRepo   *mock_domain.MockEventRepo
		clusterRepo *mock_domain.MockClusterRepo
	}
	type args struct {
		taskID string
		userID string
	}

	existing := domain.Task{
		ID:     domain.TaskID(testTaskID),
		UserID: domain.UserID(testUserID),
		Status: domain.TaskStatusPlanned,
		Title:  "title",
	}
	want := existing
	want.Status = domain.TaskStatusCompleted

	tests := []struct {
		name    string
		args    args
		setup   func(m mocks, a args)
		want    domain.Task
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "success",
			args: args{taskID: testTaskID, userID: testUserID},
			setup: func(m mocks, a args) {
				m.taskRepo.EXPECT().
					GetTaskByID(gomock.Any(), domain.TaskID(a.taskID), domain.UserID(a.userID)).
					Return(existing, nil)
				m.taskRepo.EXPECT().
					UpdateTaskStatus(gomock.Any(), domain.TaskID(a.taskID), domain.TaskStatusCompleted).
					Return(nil)
			},
			want:    want,
			wantErr: require.NoError,
		},
		{
			name: "GetTaskByID error wraps",
			args: args{taskID: testTaskID, userID: testUserID},
			setup: func(m mocks, a args) {
				m.taskRepo.EXPECT().
					GetTaskByID(gomock.Any(), domain.TaskID(a.taskID), domain.UserID(a.userID)).
					Return(domain.Task{}, assert.AnError)
			},
			want:    domain.Task{},
			wantErr: require.Error,
		},
		{
			name: "UpdateTaskStatus error wraps",
			args: args{taskID: testTaskID, userID: testUserID},
			setup: func(m mocks, a args) {
				m.taskRepo.EXPECT().
					GetTaskByID(gomock.Any(), domain.TaskID(a.taskID), domain.UserID(a.userID)).
					Return(existing, nil)
				m.taskRepo.EXPECT().
					UpdateTaskStatus(gomock.Any(), domain.TaskID(a.taskID), domain.TaskStatusCompleted).
					Return(assert.AnError)
			},
			want:    domain.Task{},
			wantErr: require.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			m := mocks{
				taskRepo:    mock_domain.NewMockTaskRepo(ctrl),
				eventRepo:   mock_domain.NewMockEventRepo(ctrl),
				clusterRepo: mock_domain.NewMockClusterRepo(ctrl),
			}
			tt.setup(m, tt.args)

			uc := tasklist.NewUseCase(m.taskRepo, m.eventRepo, m.clusterRepo, stubTxProvider{})
			got, err := uc.CompleteTask(t.Context(), tt.args.taskID, tt.args.userID)

			tt.wantErr(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestUseCase_DeleteTask(t *testing.T) {
	type mocks struct {
		taskRepo    *mock_domain.MockTaskRepo
		eventRepo   *mock_domain.MockEventRepo
		clusterRepo *mock_domain.MockClusterRepo
	}
	type args struct {
		taskID string
		userID string
	}

	clusterID := domain.ClusterID("cluster-1")
	taskWithCluster := domain.Task{
		ID:        domain.TaskID(testTaskID),
		UserID:    domain.UserID(testUserID),
		ClusterID: &clusterID,
	}
	taskWithoutCluster := domain.Task{
		ID:     domain.TaskID(testTaskID),
		UserID: domain.UserID(testUserID),
	}

	tests := []struct {
		name    string
		args    args
		txErr   error
		setup   func(m mocks, a args)
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "success with cluster",
			args: args{taskID: testTaskID, userID: testUserID},
			setup: func(m mocks, a args) {
				gomock.InOrder(
					m.taskRepo.EXPECT().
						GetTaskByID(gomock.Any(), domain.TaskID(a.taskID), domain.UserID(a.userID)).
						Return(taskWithCluster, nil),
					m.eventRepo.EXPECT().DeleteEventsByClusterID(gomock.Any(), clusterID).Return(nil),
					m.taskRepo.EXPECT().DeleteTask(gomock.Any(), domain.TaskID(a.taskID)).Return(nil),
					m.clusterRepo.EXPECT().DeleteCluster(gomock.Any(), clusterID).Return(nil),
				)
			},
			wantErr: require.NoError,
		},
		{
			name: "success without cluster",
			args: args{taskID: testTaskID, userID: testUserID},
			setup: func(m mocks, a args) {
				m.taskRepo.EXPECT().
					GetTaskByID(gomock.Any(), domain.TaskID(a.taskID), domain.UserID(a.userID)).
					Return(taskWithoutCluster, nil)
				m.taskRepo.EXPECT().DeleteTask(gomock.Any(), domain.TaskID(a.taskID)).Return(nil)
			},
			wantErr: require.NoError,
		},
		{
			name: "GetTaskByID error wraps",
			args: args{taskID: testTaskID, userID: testUserID},
			setup: func(m mocks, a args) {
				m.taskRepo.EXPECT().
					GetTaskByID(gomock.Any(), domain.TaskID(a.taskID), domain.UserID(a.userID)).
					Return(domain.Task{}, assert.AnError)
			},
			wantErr: require.Error,
		},
		{
			name: "DeleteEventsByClusterID error wraps",
			args: args{taskID: testTaskID, userID: testUserID},
			setup: func(m mocks, a args) {
				m.taskRepo.EXPECT().
					GetTaskByID(gomock.Any(), domain.TaskID(a.taskID), domain.UserID(a.userID)).
					Return(taskWithCluster, nil)
				m.eventRepo.EXPECT().DeleteEventsByClusterID(gomock.Any(), clusterID).Return(assert.AnError)
			},
			wantErr: require.Error,
		},
		{
			name: "DeleteTask error wraps",
			args: args{taskID: testTaskID, userID: testUserID},
			setup: func(m mocks, a args) {
				m.taskRepo.EXPECT().
					GetTaskByID(gomock.Any(), domain.TaskID(a.taskID), domain.UserID(a.userID)).
					Return(taskWithCluster, nil)
				m.eventRepo.EXPECT().DeleteEventsByClusterID(gomock.Any(), clusterID).Return(nil)
				m.taskRepo.EXPECT().DeleteTask(gomock.Any(), domain.TaskID(a.taskID)).Return(assert.AnError)
			},
			wantErr: require.Error,
		},
		{
			name: "DeleteCluster error wraps",
			args: args{taskID: testTaskID, userID: testUserID},
			setup: func(m mocks, a args) {
				m.taskRepo.EXPECT().
					GetTaskByID(gomock.Any(), domain.TaskID(a.taskID), domain.UserID(a.userID)).
					Return(taskWithCluster, nil)
				m.eventRepo.EXPECT().DeleteEventsByClusterID(gomock.Any(), clusterID).Return(nil)
				m.taskRepo.EXPECT().DeleteTask(gomock.Any(), domain.TaskID(a.taskID)).Return(nil)
				m.clusterRepo.EXPECT().DeleteCluster(gomock.Any(), clusterID).Return(assert.AnError)
			},
			wantErr: require.Error,
		},
		{
			name:  "tx provider error before callback",
			args:  args{taskID: testTaskID, userID: testUserID},
			txErr: assert.AnError,
			setup: func(m mocks, a args) {
				m.taskRepo.EXPECT().
					GetTaskByID(gomock.Any(), domain.TaskID(a.taskID), domain.UserID(a.userID)).
					Return(taskWithCluster, nil)
			},
			wantErr: require.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			m := mocks{
				taskRepo:    mock_domain.NewMockTaskRepo(ctrl),
				eventRepo:   mock_domain.NewMockEventRepo(ctrl),
				clusterRepo: mock_domain.NewMockClusterRepo(ctrl),
			}
			tt.setup(m, tt.args)

			uc := tasklist.NewUseCase(m.taskRepo, m.eventRepo, m.clusterRepo, stubTxProvider{err: tt.txErr})
			err := uc.DeleteTask(t.Context(), tt.args.taskID, tt.args.userID)

			tt.wantErr(t, err)
		})
	}
}

func TestUseCase_GetTasks(t *testing.T) {
	type mocks struct {
		taskRepo    *mock_domain.MockTaskRepo
		eventRepo   *mock_domain.MockEventRepo
		clusterRepo *mock_domain.MockClusterRepo
	}
	type args struct {
		params tasklist.ListTasksParams
	}

	status := domain.TaskStatusPlanned
	category := domain.TaskCategoryWork

	wantFrom := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	wantTill := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	returnedTasks := []domain.Task{{ID: "task-1"}}

	tests := []struct {
		name    string
		args    args
		setup   func(m mocks, a args)
		want    tasklist.ListTasksResult
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "success all filters",
			args: args{params: tasklist.ListTasksParams{
				UserID:   testUserID,
				Status:   &status,
				Category: &category,
				Text:     "search",
				From:     "01-06-2024",
				Till:     "31-12-2024",
				Page:     2,
				PageSize: 50,
			}},
			setup: func(m mocks, a args) {
				m.taskRepo.EXPECT().
					ListTasks(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, filter domain.TaskFilter, pagination domain.Pagination) ([]domain.Task, int, error) {
						assert.Equal(t, domain.UserID(testUserID), filter.UserID)
						assert.Equal(t, &status, filter.Status)
						assert.Equal(t, &category, filter.Category)
						assert.Equal(t, "search", filter.Text)
						require.NotNil(t, filter.From)
						assert.Equal(t, wantFrom, *filter.From)
						require.NotNil(t, filter.Till)
						assert.Equal(t, wantTill, *filter.Till)
						assert.Equal(t, 2, pagination.Page)
						assert.Equal(t, 50, pagination.PageSize)
						return returnedTasks, 1, nil
					})
			},
			want: tasklist.ListTasksResult{
				Tasks:    returnedTasks,
				Total:    1,
				Page:     2,
				PageSize: 50,
			},
			wantErr: require.NoError,
		},
		{
			name: "empty From and Till leave filter pointers nil",
			args: args{params: tasklist.ListTasksParams{
				UserID:   testUserID,
				Page:     1,
				PageSize: 10,
			}},
			setup: func(m mocks, a args) {
				m.taskRepo.EXPECT().
					ListTasks(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, filter domain.TaskFilter, pagination domain.Pagination) ([]domain.Task, int, error) {
						assert.Nil(t, filter.From)
						assert.Nil(t, filter.Till)
						assert.Equal(t, 1, pagination.Page)
						assert.Equal(t, 10, pagination.PageSize)
						return nil, 0, nil
					})
			},
			want: tasklist.ListTasksResult{
				Tasks:    nil,
				Total:    0,
				Page:     1,
				PageSize: 10,
			},
			wantErr: require.NoError,
		},
		{
			name: "invalid From format errors and skips repo",
			args: args{params: tasklist.ListTasksParams{
				UserID: testUserID,
				From:   "2024-06-01",
			}},
			setup:   func(m mocks, a args) {},
			want:    tasklist.ListTasksResult{},
			wantErr: require.Error,
		},
		{
			name: "invalid Till format errors and skips repo",
			args: args{params: tasklist.ListTasksParams{
				UserID: testUserID,
				From:   "01-06-2024",
				Till:   "not-a-date",
			}},
			setup:   func(m mocks, a args) {},
			want:    tasklist.ListTasksResult{},
			wantErr: require.Error,
		},
		{
			name: "repo error wraps",
			args: args{params: tasklist.ListTasksParams{
				UserID:   testUserID,
				Page:     1,
				PageSize: 10,
			}},
			setup: func(m mocks, a args) {
				m.taskRepo.EXPECT().
					ListTasks(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, 0, assert.AnError)
			},
			want:    tasklist.ListTasksResult{},
			wantErr: require.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			m := mocks{
				taskRepo:    mock_domain.NewMockTaskRepo(ctrl),
				eventRepo:   mock_domain.NewMockEventRepo(ctrl),
				clusterRepo: mock_domain.NewMockClusterRepo(ctrl),
			}
			tt.setup(m, tt.args)

			uc := tasklist.NewUseCase(m.taskRepo, m.eventRepo, m.clusterRepo, stubTxProvider{})
			got, err := uc.GetTasks(t.Context(), tt.args.params)

			tt.wantErr(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

type stubTxProvider struct{ err error }

func (s stubTxProvider) RunWithTx(ctx context.Context, _ tx.Isolation, op func(context.Context) error) error {
	if s.err != nil {
		return s.err
	}
	return op(ctx)
}
