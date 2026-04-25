package grpc

import (
	"context"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/Doremi203/personage/backend/libs/go/log"
	"github.com/Doremi203/personage/backend/libs/go/token"
	taskspb "github.com/Doremi203/personage/backend/tasker/gen/api/tasks"
	"github.com/Doremi203/personage/backend/tasker/internal/domain"
	"github.com/Doremi203/personage/backend/tasker/internal/usecase/tasklist"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func NewTasksService(
	listUseCase *tasklist.UseCase,
	logger log.Logger,
) *tasksService {
	return &tasksService{
		listUseCase: listUseCase,
		logger:      logger,
	}
}

type tasksService struct {
	listUseCase *tasklist.UseCase
	logger      log.Logger
	taskspb.UnimplementedTasksServer
}

func (s *tasksService) RegisterToGateway(
	ctx context.Context,
	mux *runtime.ServeMux,
	endpoint string,
	opts []grpc.DialOption,
) error {
	return taskspb.RegisterTasksHandlerFromEndpoint(ctx, mux, endpoint, opts)
}

func (s *tasksService) RegisterToServer(gRPC *grpc.Server) {
	taskspb.RegisterTasksServer(gRPC, s)
}

func (s *tasksService) ListTasksV1(
	ctx context.Context,
	req *taskspb.ListTasksV1Request,
) (*taskspb.ListTasksV1Response, error) {
	t, ok := token.FromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing token")
	}

	if err := req.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	var statusFilter *domain.TaskStatus
	if s := protoStatusFilterToDomain(req.GetStatus()); s != "" {
		statusFilter = &s
	}

	var categoryFilter *domain.TaskCategory
	if c := protoCategoryFilterToDomain(req.GetCategory()); c != "" {
		categoryFilter = &c
	}

	result, err := s.listUseCase.GetTasks(ctx, tasklist.ListTasksParams{
		UserID:   t.GetUserID().String(),
		Status:   statusFilter,
		Category: categoryFilter,
		Text:     req.GetText(),
		From:     req.GetFrom(),
		Till:     req.GetTill(),
		Page:     req.GetPage(),
		PageSize: req.GetPageSize(),
	})
	if err != nil {
		return nil, errors.WrapFail(err, "get tasks")
	}

	protoTasks := make([]*taskspb.TaskItem, 0, len(result.Tasks))
	for _, task := range result.Tasks {
		protoTasks = append(protoTasks, domainTaskToProto(task))
	}

	return &taskspb.ListTasksV1Response{
		Tasks:    protoTasks,
		Total:    int32(result.Total),
		Page:     int32(result.Page),
		PageSize: int32(result.PageSize),
	}, nil
}

func (s *tasksService) GetTaskV1(
	ctx context.Context,
	req *taskspb.GetTaskV1Request,
) (*taskspb.GetTaskV1Response, error) {
	t, ok := token.FromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing token")
	}

	if err := req.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	task, err := s.listUseCase.GetTask(ctx, req.GetId(), t.GetUserID().String())
	if err != nil {
		if errors.Is(err, domain.ErrTaskNotFound) {
			return nil, status.Error(codes.NotFound, "task not found")
		}
		return nil, errors.WrapFail(err, "get task")
	}

	return &taskspb.GetTaskV1Response{
		Task: domainTaskToProto(task),
	}, nil
}

func (s *tasksService) UpdateTaskV1(
	ctx context.Context,
	req *taskspb.UpdateTaskV1Request,
) (*taskspb.UpdateTaskV1Response, error) {
	t, ok := token.FromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing token")
	}

	if err := req.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	update := domain.TaskUpdate{}

	if title := req.GetTitle(); title != "" {
		update.Title = &title
	}

	if desc := req.GetDescription(); desc != "" {
		update.Description = &desc
	}

	if req.GetStartTime() != nil {
		st := req.GetStartTime().AsTime()
		update.StartTime = &st
	}

	if req.GetEndTime() != nil {
		et := req.GetEndTime().AsTime()
		update.EndTime = &et
	}

	if req.GetCategory() != taskspb.TaskCategory_TASK_CATEGORY_UNSPECIFIED {
		cat := protoCategoryToDomain(req.GetCategory())
		update.Category = &cat
	}

	task, err := s.listUseCase.UpdateTask(ctx, req.GetId(), t.GetUserID().String(), update)
	if err != nil {
		if errors.Is(err, domain.ErrTaskNotFound) {
			return nil, status.Error(codes.NotFound, "task not found")
		}
		return nil, errors.WrapFail(err, "update task")
	}

	return &taskspb.UpdateTaskV1Response{
		Task: domainTaskToProto(task),
	}, nil
}

func (s *tasksService) PostponeTaskV1(
	ctx context.Context,
	req *taskspb.PostponeTaskV1Request,
) (*taskspb.PostponeTaskV1Response, error) {
	t, ok := token.FromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing token")
	}

	if err := req.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	task, err := s.listUseCase.PostponeTask(ctx, req.GetId(), t.GetUserID().String())
	if err != nil {
		if errors.Is(err, domain.ErrTaskNotFound) {
			return nil, status.Error(codes.NotFound, "task not found")
		}
		return nil, errors.WrapFail(err, "postpone task")
	}

	return &taskspb.PostponeTaskV1Response{
		Task: domainTaskToProto(task),
	}, nil
}

func (s *tasksService) DeleteTaskV1(
	ctx context.Context,
	req *taskspb.DeleteTaskV1Request,
) (*taskspb.DeleteTaskV1Response, error) {
	t, ok := token.FromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing token")
	}

	if err := req.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	err := s.listUseCase.DeleteTask(ctx, req.GetId(), t.GetUserID().String())
	if err != nil {
		if errors.Is(err, domain.ErrTaskNotFound) {
			return nil, status.Error(codes.NotFound, "task not found")
		}
		return nil, errors.WrapFail(err, "delete task")
	}

	return &taskspb.DeleteTaskV1Response{}, nil
}

func (s *tasksService) CompleteTaskV1(
	ctx context.Context,
	req *taskspb.CompleteTaskV1Request,
) (*taskspb.CompleteTaskV1Response, error) {
	t, ok := token.FromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing token")
	}

	if err := req.ValidateAll(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	task, err := s.listUseCase.CompleteTask(ctx, req.GetId(), t.GetUserID().String())
	if err != nil {
		if errors.Is(err, domain.ErrTaskNotFound) {
			return nil, status.Error(codes.NotFound, "task not found")
		}
		return nil, errors.WrapFail(err, "complete task")
	}

	return &taskspb.CompleteTaskV1Response{
		Task: domainTaskToProto(task),
	}, nil
}

func protoStatusFilterToDomain(s taskspb.TaskStatusFilter) domain.TaskStatus {
	switch s {
	case taskspb.TaskStatusFilter_TASK_STATUS_FILTER_UNPLANNED:
		return domain.TaskStatusUnplanned
	case taskspb.TaskStatusFilter_TASK_STATUS_FILTER_PLANNED:
		return domain.TaskStatusPlanned
	case taskspb.TaskStatusFilter_TASK_STATUS_FILTER_COMPLETED:
		return domain.TaskStatusCompleted
	default:
		return ""
	}
}

func protoCategoryFilterToDomain(c taskspb.TaskCategoryFilter) domain.TaskCategory {
	switch c {
	case taskspb.TaskCategoryFilter_TASK_CATEGORY_FILTER_WORK:
		return domain.TaskCategoryWork
	case taskspb.TaskCategoryFilter_TASK_CATEGORY_FILTER_STUDY:
		return domain.TaskCategoryStudy
	case taskspb.TaskCategoryFilter_TASK_CATEGORY_FILTER_PERSONAL:
		return domain.TaskCategoryPersonal
	default:
		return ""
	}
}

func protoCategoryToDomain(c taskspb.TaskCategory) domain.TaskCategory {
	switch c {
	case taskspb.TaskCategory_TASK_CATEGORY_WORK:
		return domain.TaskCategoryWork
	case taskspb.TaskCategory_TASK_CATEGORY_STUDY:
		return domain.TaskCategoryStudy
	case taskspb.TaskCategory_TASK_CATEGORY_PERSONAL:
		return domain.TaskCategoryPersonal
	default:
		return ""
	}
}

func domainPriorityToProto(p domain.TaskPriority) taskspb.TaskPriority {
	switch p {
	case domain.TaskPriorityLow:
		return taskspb.TaskPriority_TASK_PRIORITY_LOW
	case domain.TaskPriorityMid:
		return taskspb.TaskPriority_TASK_PRIORITY_MID
	case domain.TaskPriorityHigh:
		return taskspb.TaskPriority_TASK_PRIORITY_HIGH
	default:
		return taskspb.TaskPriority_TASK_PRIORITY_UNSPECIFIED
	}
}

func domainStatusToProto(s domain.TaskStatus) taskspb.TaskStatus {
	switch s {
	case domain.TaskStatusUnplanned:
		return taskspb.TaskStatus_TASK_STATUS_UNPLANNED
	case domain.TaskStatusPlanned:
		return taskspb.TaskStatus_TASK_STATUS_PLANNED
	case domain.TaskStatusCompleted:
		return taskspb.TaskStatus_TASK_STATUS_COMPLETED
	default:
		return taskspb.TaskStatus_TASK_STATUS_UNSPECIFIED
	}
}

func domainCategoryToProto(c domain.TaskCategory) taskspb.TaskCategory {
	switch c {
	case domain.TaskCategoryWork:
		return taskspb.TaskCategory_TASK_CATEGORY_WORK
	case domain.TaskCategoryStudy:
		return taskspb.TaskCategory_TASK_CATEGORY_STUDY
	case domain.TaskCategoryPersonal:
		return taskspb.TaskCategory_TASK_CATEGORY_PERSONAL
	default:
		return taskspb.TaskCategory_TASK_CATEGORY_UNSPECIFIED
	}
}

func domainTaskToProto(task domain.Task) *taskspb.TaskItem {
	item := &taskspb.TaskItem{
		Id:          task.ID.String(),
		Title:       task.Title,
		Description: task.Description,
		Priority:    domainPriorityToProto(domain.PriorityFromInt(task.Priority)),
		Status:      domainStatusToProto(task.Status),
		Category:    domainCategoryToProto(task.Category),
		UpdatedAt:   timestamppb.New(task.UpdatedAt),
		CreatedAt:   timestamppb.New(task.CreatedAt),
	}

	if task.StartTime != nil {
		item.StartTime = timestamppb.New(*task.StartTime)
	}

	if task.EndTime != nil {
		item.EndTime = timestamppb.New(*task.EndTime)
	}

	if task.Deadline != nil {
		item.Deadline = timestamppb.New(*task.Deadline)
	}

	return item
}
