package grpc_test

import (
	"context"
	"testing"
	"time"

	"github.com/Doremi203/personage/backend/libs/go/log"
	"github.com/Doremi203/personage/backend/libs/go/token"
	"github.com/Doremi203/personage/backend/libs/go/tx"
	taskspb "github.com/Doremi203/personage/backend/tasker/gen/api/tasks"
	"github.com/Doremi203/personage/backend/tasker/internal/domain"
	mock_domain "github.com/Doremi203/personage/backend/tasker/internal/domain/mock"
	taskergrpc "github.com/Doremi203/personage/backend/tasker/internal/grpc"
	"github.com/Doremi203/personage/backend/tasker/internal/usecase/tasklist"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var grpcUserID = uuid.MustParse("11111111-1111-1111-1111-111111111111")

const grpcTaskID = "task-1"

type tasksDeps struct {
	taskRepo    *mock_domain.MockTaskRepo
	eventRepo   *mock_domain.MockEventRepo
	clusterRepo *mock_domain.MockClusterRepo
}

func newTasksService(t *testing.T, txErr error) (taskspb.TasksServer, tasksDeps) {
	t.Helper()
	ctrl := gomock.NewController(t)
	d := tasksDeps{
		taskRepo:    mock_domain.NewMockTaskRepo(ctrl),
		eventRepo:   mock_domain.NewMockEventRepo(ctrl),
		clusterRepo: mock_domain.NewMockClusterRepo(ctrl),
	}
	uc := tasklist.NewUseCase(d.taskRepo, d.eventRepo, d.clusterRepo, stubTxProvider{err: txErr})
	return taskergrpc.NewTasksService(uc, log.Stub{}), d
}

func authedCtx(t *testing.T) context.Context {
	t.Helper()
	return token.ContextWithToken(t.Context(), token.Token{UserID: grpcUserID})
}

func TestTasksService_ListTasksV1(t *testing.T) {
	type args struct {
		ctx context.Context
		req *taskspb.ListTasksV1Request
	}
	tests := []struct {
		name       string
		args       func(t *testing.T) args
		setup      func(d tasksDeps)
		wantCode   codes.Code
		wantTotal  int32
		wantTaskID string
	}{
		{
			name: "success returns mapped tasks",
			args: func(t *testing.T) args {
				return args{
					ctx: authedCtx(t),
					req: &taskspb.ListTasksV1Request{
						Status:   taskspb.TaskStatusFilter_TASK_STATUS_FILTER_PLANNED,
						Category: taskspb.TaskCategoryFilter_TASK_CATEGORY_FILTER_WORK,
						Page:     1,
						PageSize: 10,
					},
				}
			},
			setup: func(d tasksDeps) {
				d.taskRepo.EXPECT().
					ListTasks(gomock.Any(), gomock.Any(), gomock.Any()).
					Return([]domain.Task{{
						ID:       domain.TaskID(grpcTaskID),
						UserID:   domain.UserID(grpcUserID.String()),
						Title:    "t",
						Status:   domain.TaskStatusPlanned,
						Category: domain.TaskCategoryWork,
						Priority: 5,
					}}, 1, nil)
			},
			wantCode:   codes.OK,
			wantTotal:  1,
			wantTaskID: grpcTaskID,
		},
		{
			name: "missing token returns Unauthenticated",
			args: func(t *testing.T) args {
				return args{ctx: t.Context(), req: &taskspb.ListTasksV1Request{Page: 1, PageSize: 10}}
			},
			setup:    func(d tasksDeps) {},
			wantCode: codes.Unauthenticated,
		},
		{
			name: "invalid request returns InvalidArgument",
			args: func(t *testing.T) args {
				return args{ctx: authedCtx(t), req: &taskspb.ListTasksV1Request{PageSize: 0}}
			},
			setup:    func(d tasksDeps) {},
			wantCode: codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, d := newTasksService(t, nil)
			tt.setup(d)
			a := tt.args(t)

			resp, err := svc.ListTasksV1(a.ctx, a.req)

			if tt.wantCode == codes.OK {
				require.NoError(t, err)
				require.NotNil(t, resp)
				assert.Equal(t, tt.wantTotal, resp.GetTotal())
				require.Len(t, resp.GetTasks(), 1)
				assert.Equal(t, tt.wantTaskID, resp.GetTasks()[0].GetId())
				return
			}
			require.Error(t, err)
			assert.Equal(t, tt.wantCode, status.Code(err))
		})
	}
}

func TestTasksService_GetTaskV1(t *testing.T) {
	type args struct {
		ctx context.Context
		req *taskspb.GetTaskV1Request
	}
	deadline := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	startTime := time.Date(2026, 5, 1, 9, 0, 0, 0, time.UTC)
	endTime := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		args     func(t *testing.T) args
		setup    func(d tasksDeps)
		wantCode codes.Code
	}{
		{
			name: "success maps optional times",
			args: func(t *testing.T) args {
				return args{ctx: authedCtx(t), req: &taskspb.GetTaskV1Request{Id: grpcTaskID}}
			},
			setup: func(d tasksDeps) {
				d.taskRepo.EXPECT().
					GetTaskByID(gomock.Any(), domain.TaskID(grpcTaskID), domain.UserID(grpcUserID.String())).
					Return(domain.Task{
						ID:        domain.TaskID(grpcTaskID),
						Title:     "t",
						Status:    domain.TaskStatusPlanned,
						Category:  domain.TaskCategoryStudy,
						Priority:  9,
						Deadline:  &deadline,
						StartTime: &startTime,
						EndTime:   &endTime,
					}, nil)
			},
			wantCode: codes.OK,
		},
		{
			name: "missing token",
			args: func(t *testing.T) args {
				return args{ctx: t.Context(), req: &taskspb.GetTaskV1Request{Id: grpcTaskID}}
			},
			setup:    func(d tasksDeps) {},
			wantCode: codes.Unauthenticated,
		},
		{
			name: "invalid id",
			args: func(t *testing.T) args {
				return args{ctx: authedCtx(t), req: &taskspb.GetTaskV1Request{Id: ""}}
			},
			setup:    func(d tasksDeps) {},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "not found maps to NotFound",
			args: func(t *testing.T) args {
				return args{ctx: authedCtx(t), req: &taskspb.GetTaskV1Request{Id: grpcTaskID}}
			},
			setup: func(d tasksDeps) {
				d.taskRepo.EXPECT().
					GetTaskByID(gomock.Any(), domain.TaskID(grpcTaskID), domain.UserID(grpcUserID.String())).
					Return(domain.Task{}, domain.ErrTaskNotFound)
			},
			wantCode: codes.NotFound,
		},
		{
			name: "repo error returns Unknown",
			args: func(t *testing.T) args {
				return args{ctx: authedCtx(t), req: &taskspb.GetTaskV1Request{Id: grpcTaskID}}
			},
			setup: func(d tasksDeps) {
				d.taskRepo.EXPECT().
					GetTaskByID(gomock.Any(), domain.TaskID(grpcTaskID), domain.UserID(grpcUserID.String())).
					Return(domain.Task{}, assert.AnError)
			},
			wantCode: codes.Unknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, d := newTasksService(t, nil)
			tt.setup(d)
			a := tt.args(t)

			resp, err := svc.GetTaskV1(a.ctx, a.req)

			if tt.wantCode == codes.OK {
				require.NoError(t, err)
				require.NotNil(t, resp)
				assert.Equal(t, grpcTaskID, resp.GetTask().GetId())
				assert.Equal(t, taskspb.TaskPriority_TASK_PRIORITY_HIGH, resp.GetTask().GetPriority())
				assert.Equal(t, taskspb.TaskCategory_TASK_CATEGORY_STUDY, resp.GetTask().GetCategory())
				assert.Equal(t, timestamppb.New(deadline).AsTime(), resp.GetTask().GetDeadline().AsTime())
				assert.Equal(t, timestamppb.New(startTime).AsTime(), resp.GetTask().GetStartTime().AsTime())
				assert.Equal(t, timestamppb.New(endTime).AsTime(), resp.GetTask().GetEndTime().AsTime())
				return
			}
			require.Error(t, err)
			assert.Equal(t, tt.wantCode, status.Code(err))
		})
	}
}

func TestTasksService_UpdateTaskV1(t *testing.T) {
	startTime := time.Date(2026, 5, 1, 9, 0, 0, 0, time.UTC)
	endTime := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)

	type args struct {
		ctx context.Context
		req *taskspb.UpdateTaskV1Request
	}
	tests := []struct {
		name     string
		args     func(t *testing.T) args
		setup    func(d tasksDeps)
		wantCode codes.Code
	}{
		{
			name: "success applies all fields",
			args: func(t *testing.T) args {
				return args{ctx: authedCtx(t), req: &taskspb.UpdateTaskV1Request{
					Id:          grpcTaskID,
					Title:       new("new"),
					Description: new("desc"),
					StartTime:   timestamppb.New(startTime),
					EndTime:     timestamppb.New(endTime),
					Category:    taskspb.TaskCategory_TASK_CATEGORY_PERSONAL,
				}}
			},
			setup: func(d tasksDeps) {
				d.taskRepo.EXPECT().
					UpdateTask(gomock.Any(), domain.TaskID(grpcTaskID), domain.UserID(grpcUserID.String()), gomock.Any()).
					DoAndReturn(func(_ context.Context, _ domain.TaskID, _ domain.UserID, upd domain.TaskUpdate) (domain.Task, error) {
						require.NotNil(t, upd.Title)
						assert.Equal(t, "new", *upd.Title)
						require.NotNil(t, upd.Description)
						assert.Equal(t, "desc", *upd.Description)
						require.NotNil(t, upd.StartTime)
						assert.Equal(t, startTime, *upd.StartTime)
						require.NotNil(t, upd.EndTime)
						assert.Equal(t, endTime, *upd.EndTime)
						require.NotNil(t, upd.Category)
						assert.Equal(t, domain.TaskCategoryPersonal, *upd.Category)
						return domain.Task{ID: domain.TaskID(grpcTaskID), Title: "new"}, nil
					})
			},
			wantCode: codes.OK,
		},
		{
			name: "success without optional fields keeps update zero",
			args: func(t *testing.T) args {
				return args{ctx: authedCtx(t), req: &taskspb.UpdateTaskV1Request{Id: grpcTaskID}}
			},
			setup: func(d tasksDeps) {
				d.taskRepo.EXPECT().
					UpdateTask(gomock.Any(), domain.TaskID(grpcTaskID), domain.UserID(grpcUserID.String()), domain.TaskUpdate{}).
					Return(domain.Task{ID: domain.TaskID(grpcTaskID)}, nil)
			},
			wantCode: codes.OK,
		},
		{
			name: "missing token",
			args: func(t *testing.T) args {
				return args{ctx: t.Context(), req: &taskspb.UpdateTaskV1Request{Id: grpcTaskID}}
			},
			setup:    func(d tasksDeps) {},
			wantCode: codes.Unauthenticated,
		},
		{
			name: "invalid request",
			args: func(t *testing.T) args {
				return args{ctx: authedCtx(t), req: &taskspb.UpdateTaskV1Request{Id: ""}}
			},
			setup:    func(d tasksDeps) {},
			wantCode: codes.InvalidArgument,
		},
		{
			name: "not found maps to NotFound",
			args: func(t *testing.T) args {
				return args{ctx: authedCtx(t), req: &taskspb.UpdateTaskV1Request{Id: grpcTaskID, Title: new("x")}}
			},
			setup: func(d tasksDeps) {
				d.taskRepo.EXPECT().
					UpdateTask(gomock.Any(), domain.TaskID(grpcTaskID), domain.UserID(grpcUserID.String()), gomock.Any()).
					Return(domain.Task{}, domain.ErrTaskNotFound)
			},
			wantCode: codes.NotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, d := newTasksService(t, nil)
			tt.setup(d)
			a := tt.args(t)

			resp, err := svc.UpdateTaskV1(a.ctx, a.req)

			if tt.wantCode == codes.OK {
				require.NoError(t, err)
				require.NotNil(t, resp)
				assert.Equal(t, grpcTaskID, resp.GetTask().GetId())
				return
			}
			require.Error(t, err)
			assert.Equal(t, tt.wantCode, status.Code(err))
		})
	}
}

func TestTasksService_PostponeTaskV1(t *testing.T) {
	tests := []struct {
		name     string
		ctxFn    func(t *testing.T) context.Context
		req      *taskspb.PostponeTaskV1Request
		setup    func(d tasksDeps)
		wantCode codes.Code
	}{
		{
			name:  "success",
			ctxFn: authedCtx,
			req:   &taskspb.PostponeTaskV1Request{Id: grpcTaskID},
			setup: func(d tasksDeps) {
				gomock.InOrder(
					d.taskRepo.EXPECT().
						GetTaskByID(gomock.Any(), domain.TaskID(grpcTaskID), domain.UserID(grpcUserID.String())).
						Return(domain.Task{ID: domain.TaskID(grpcTaskID)}, nil),
					d.taskRepo.EXPECT().
						UpdateTaskStatus(gomock.Any(), domain.TaskID(grpcTaskID), domain.TaskStatusUnplanned).
						Return(nil),
				)
			},
			wantCode: codes.OK,
		},
		{
			name:     "missing token",
			ctxFn:    func(t *testing.T) context.Context { return t.Context() },
			req:      &taskspb.PostponeTaskV1Request{Id: grpcTaskID},
			setup:    func(d tasksDeps) {},
			wantCode: codes.Unauthenticated,
		},
		{
			name:     "invalid id",
			ctxFn:    authedCtx,
			req:      &taskspb.PostponeTaskV1Request{Id: ""},
			setup:    func(d tasksDeps) {},
			wantCode: codes.InvalidArgument,
		},
		{
			name:  "not found",
			ctxFn: authedCtx,
			req:   &taskspb.PostponeTaskV1Request{Id: grpcTaskID},
			setup: func(d tasksDeps) {
				d.taskRepo.EXPECT().
					GetTaskByID(gomock.Any(), domain.TaskID(grpcTaskID), domain.UserID(grpcUserID.String())).
					Return(domain.Task{}, domain.ErrTaskNotFound)
			},
			wantCode: codes.NotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, d := newTasksService(t, nil)
			tt.setup(d)

			resp, err := svc.PostponeTaskV1(tt.ctxFn(t), tt.req)

			if tt.wantCode == codes.OK {
				require.NoError(t, err)
				assert.Equal(t, grpcTaskID, resp.GetTask().GetId())
				return
			}
			require.Error(t, err)
			assert.Equal(t, tt.wantCode, status.Code(err))
		})
	}
}

func TestTasksService_CompleteTaskV1(t *testing.T) {
	tests := []struct {
		name     string
		ctxFn    func(t *testing.T) context.Context
		req      *taskspb.CompleteTaskV1Request
		setup    func(d tasksDeps)
		wantCode codes.Code
	}{
		{
			name:  "success",
			ctxFn: authedCtx,
			req:   &taskspb.CompleteTaskV1Request{Id: grpcTaskID},
			setup: func(d tasksDeps) {
				gomock.InOrder(
					d.taskRepo.EXPECT().
						GetTaskByID(gomock.Any(), domain.TaskID(grpcTaskID), domain.UserID(grpcUserID.String())).
						Return(domain.Task{ID: domain.TaskID(grpcTaskID)}, nil),
					d.taskRepo.EXPECT().
						UpdateTaskStatus(gomock.Any(), domain.TaskID(grpcTaskID), domain.TaskStatusCompleted).
						Return(nil),
				)
			},
			wantCode: codes.OK,
		},
		{
			name:     "missing token",
			ctxFn:    func(t *testing.T) context.Context { return t.Context() },
			req:      &taskspb.CompleteTaskV1Request{Id: grpcTaskID},
			setup:    func(d tasksDeps) {},
			wantCode: codes.Unauthenticated,
		},
		{
			name:     "invalid id",
			ctxFn:    authedCtx,
			req:      &taskspb.CompleteTaskV1Request{Id: ""},
			setup:    func(d tasksDeps) {},
			wantCode: codes.InvalidArgument,
		},
		{
			name:  "not found",
			ctxFn: authedCtx,
			req:   &taskspb.CompleteTaskV1Request{Id: grpcTaskID},
			setup: func(d tasksDeps) {
				d.taskRepo.EXPECT().
					GetTaskByID(gomock.Any(), domain.TaskID(grpcTaskID), domain.UserID(grpcUserID.String())).
					Return(domain.Task{}, domain.ErrTaskNotFound)
			},
			wantCode: codes.NotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, d := newTasksService(t, nil)
			tt.setup(d)

			resp, err := svc.CompleteTaskV1(tt.ctxFn(t), tt.req)

			if tt.wantCode == codes.OK {
				require.NoError(t, err)
				assert.Equal(t, grpcTaskID, resp.GetTask().GetId())
				return
			}
			require.Error(t, err)
			assert.Equal(t, tt.wantCode, status.Code(err))
		})
	}
}

func TestTasksService_DeleteTaskV1(t *testing.T) {
	clusterID := domain.ClusterID("cluster-1")

	tests := []struct {
		name     string
		ctxFn    func(t *testing.T) context.Context
		req      *taskspb.DeleteTaskV1Request
		setup    func(d tasksDeps)
		wantCode codes.Code
	}{
		{
			name:  "success without cluster",
			ctxFn: authedCtx,
			req:   &taskspb.DeleteTaskV1Request{Id: grpcTaskID},
			setup: func(d tasksDeps) {
				d.taskRepo.EXPECT().
					GetTaskByID(gomock.Any(), domain.TaskID(grpcTaskID), domain.UserID(grpcUserID.String())).
					Return(domain.Task{ID: domain.TaskID(grpcTaskID)}, nil)
				d.taskRepo.EXPECT().DeleteTask(gomock.Any(), domain.TaskID(grpcTaskID)).Return(nil)
			},
			wantCode: codes.OK,
		},
		{
			name:  "success with cluster cascades to events and cluster",
			ctxFn: authedCtx,
			req:   &taskspb.DeleteTaskV1Request{Id: grpcTaskID},
			setup: func(d tasksDeps) {
				gomock.InOrder(
					d.taskRepo.EXPECT().
						GetTaskByID(gomock.Any(), domain.TaskID(grpcTaskID), domain.UserID(grpcUserID.String())).
						Return(domain.Task{ID: domain.TaskID(grpcTaskID), ClusterID: &clusterID}, nil),
					d.eventRepo.EXPECT().DeleteEventsByClusterID(gomock.Any(), clusterID).Return(nil),
					d.taskRepo.EXPECT().DeleteTask(gomock.Any(), domain.TaskID(grpcTaskID)).Return(nil),
					d.clusterRepo.EXPECT().DeleteCluster(gomock.Any(), clusterID).Return(nil),
				)
			},
			wantCode: codes.OK,
		},
		{
			name:     "missing token",
			ctxFn:    func(t *testing.T) context.Context { return t.Context() },
			req:      &taskspb.DeleteTaskV1Request{Id: grpcTaskID},
			setup:    func(d tasksDeps) {},
			wantCode: codes.Unauthenticated,
		},
		{
			name:     "invalid id",
			ctxFn:    authedCtx,
			req:      &taskspb.DeleteTaskV1Request{Id: ""},
			setup:    func(d tasksDeps) {},
			wantCode: codes.InvalidArgument,
		},
		{
			name:  "not found",
			ctxFn: authedCtx,
			req:   &taskspb.DeleteTaskV1Request{Id: grpcTaskID},
			setup: func(d tasksDeps) {
				d.taskRepo.EXPECT().
					GetTaskByID(gomock.Any(), domain.TaskID(grpcTaskID), domain.UserID(grpcUserID.String())).
					Return(domain.Task{}, domain.ErrTaskNotFound)
			},
			wantCode: codes.NotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, d := newTasksService(t, nil)
			tt.setup(d)

			_, err := svc.DeleteTaskV1(tt.ctxFn(t), tt.req)

			if tt.wantCode == codes.OK {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Equal(t, tt.wantCode, status.Code(err))
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
