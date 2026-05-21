package admin_test

import (
	"context"
	"testing"

	"github.com/Doremi203/personage/backend/tasker/internal/domain"
	mock_domain "github.com/Doremi203/personage/backend/tasker/internal/domain/mock"
	"github.com/Doremi203/personage/backend/tasker/internal/usecase/admin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const (
	testTaskID = "task-1"
	testUserID = "user-1"
)

func TestUseCase_ListTasks(t *testing.T) {
	type mocks struct {
		taskRepo       *mock_domain.MockTaskRepo
		moderationRepo *mock_domain.MockManualModerationRepo
	}

	wantTasks := []domain.Task{{ID: "task-1", IsApproved: false}, {ID: "task-2", IsApproved: true}}

	tests := []struct {
		name    string
		setup   func(m mocks)
		want    []domain.Task
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "passes IncludeUnapproved filter",
			setup: func(m mocks) {
				m.taskRepo.EXPECT().
					ListTasks(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, filter domain.TaskFilter, _ domain.Pagination) ([]domain.Task, int, error) {
						assert.Equal(t, domain.UserID(testUserID), filter.UserID)
						assert.True(t, filter.IncludeUnapproved)
						return wantTasks, len(wantTasks), nil
					})
			},
			want:    wantTasks,
			wantErr: require.NoError,
		},
		{
			name: "repo error wraps",
			setup: func(m mocks) {
				m.taskRepo.EXPECT().
					ListTasks(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, 0, assert.AnError)
			},
			want:    nil,
			wantErr: require.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			m := mocks{
				taskRepo:       mock_domain.NewMockTaskRepo(ctrl),
				moderationRepo: mock_domain.NewMockManualModerationRepo(ctrl),
			}
			tt.setup(m)

			uc := admin.NewUseCase(m.taskRepo, m.moderationRepo)
			got, err := uc.ListTasks(t.Context(), testUserID)

			tt.wantErr(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestUseCase_Approve(t *testing.T) {
	type mocks struct {
		taskRepo       *mock_domain.MockTaskRepo
		moderationRepo *mock_domain.MockManualModerationRepo
	}

	approvedTask := domain.Task{ID: testTaskID, IsApproved: true}

	tests := []struct {
		name    string
		setup   func(m mocks)
		want    domain.Task
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "sets IsApproved=true via UpdateTask",
			setup: func(m mocks) {
				m.taskRepo.EXPECT().
					UpdateTask(gomock.Any(), domain.TaskID(testTaskID), domain.UserID(testUserID), gomock.Any()).
					DoAndReturn(func(_ context.Context, _ domain.TaskID, _ domain.UserID, update domain.TaskUpdate) (domain.Task, error) {
						require.NotNil(t, update.IsApproved)
						assert.True(t, *update.IsApproved)
						return approvedTask, nil
					})
			},
			want:    approvedTask,
			wantErr: require.NoError,
		},
		{
			name: "repo error wraps",
			setup: func(m mocks) {
				m.taskRepo.EXPECT().
					UpdateTask(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
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
				taskRepo:       mock_domain.NewMockTaskRepo(ctrl),
				moderationRepo: mock_domain.NewMockManualModerationRepo(ctrl),
			}
			tt.setup(m)

			uc := admin.NewUseCase(m.taskRepo, m.moderationRepo)
			got, err := uc.Approve(t.Context(), testTaskID, testUserID)

			tt.wantErr(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestUseCase_SetUserModeration(t *testing.T) {
	type mocks struct {
		taskRepo       *mock_domain.MockTaskRepo
		moderationRepo *mock_domain.MockManualModerationRepo
	}
	type args struct {
		enabled bool
	}

	tests := []struct {
		name    string
		args    args
		setup   func(m mocks)
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "enabled calls AddUser",
			args: args{enabled: true},
			setup: func(m mocks) {
				m.moderationRepo.EXPECT().AddUser(gomock.Any(), domain.UserID(testUserID)).Return(nil)
			},
			wantErr: require.NoError,
		},
		{
			name: "disabled calls RemoveUser",
			args: args{enabled: false},
			setup: func(m mocks) {
				m.moderationRepo.EXPECT().RemoveUser(gomock.Any(), domain.UserID(testUserID)).Return(nil)
			},
			wantErr: require.NoError,
		},
		{
			name: "AddUser error wraps",
			args: args{enabled: true},
			setup: func(m mocks) {
				m.moderationRepo.EXPECT().AddUser(gomock.Any(), domain.UserID(testUserID)).Return(assert.AnError)
			},
			wantErr: require.Error,
		},
		{
			name: "RemoveUser error wraps",
			args: args{enabled: false},
			setup: func(m mocks) {
				m.moderationRepo.EXPECT().RemoveUser(gomock.Any(), domain.UserID(testUserID)).Return(assert.AnError)
			},
			wantErr: require.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			m := mocks{
				taskRepo:       mock_domain.NewMockTaskRepo(ctrl),
				moderationRepo: mock_domain.NewMockManualModerationRepo(ctrl),
			}
			tt.setup(m)

			uc := admin.NewUseCase(m.taskRepo, m.moderationRepo)
			err := uc.SetUserModeration(t.Context(), testUserID, tt.args.enabled)

			tt.wantErr(t, err)
		})
	}
}

func TestUseCase_ListModeratedUsers(t *testing.T) {
	type mocks struct {
		taskRepo       *mock_domain.MockTaskRepo
		moderationRepo *mock_domain.MockManualModerationRepo
	}

	wantIDs := []domain.UserID{"u-1", "u-2"}

	tests := []struct {
		name    string
		setup   func(m mocks)
		want    []domain.UserID
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "returns repo result",
			setup: func(m mocks) {
				m.moderationRepo.EXPECT().ListUsers(gomock.Any()).Return(wantIDs, nil)
			},
			want:    wantIDs,
			wantErr: require.NoError,
		},
		{
			name: "repo error wraps",
			setup: func(m mocks) {
				m.moderationRepo.EXPECT().ListUsers(gomock.Any()).Return(nil, assert.AnError)
			},
			want:    nil,
			wantErr: require.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			m := mocks{
				taskRepo:       mock_domain.NewMockTaskRepo(ctrl),
				moderationRepo: mock_domain.NewMockManualModerationRepo(ctrl),
			}
			tt.setup(m)

			uc := admin.NewUseCase(m.taskRepo, m.moderationRepo)
			got, err := uc.ListModeratedUsers(t.Context())

			tt.wantErr(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
