package taskgeneration

import (
	"context"
	"time"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/Doremi203/personage/backend/tasker/internal/domain"
	"github.com/google/uuid"
)

type UseCase struct {
	clusterRepo       domain.ClusterRepo
	eventRepo         domain.EventRepo
	taskRepo          domain.TaskRepo
	taskGenService    domain.TaskGenerationService
	maxEventCount     int
	inactivityTimeout time.Duration
}

func NewUseCase(
	clusterRepo domain.ClusterRepo,
	eventRepo domain.EventRepo,
	taskRepo domain.TaskRepo,
	taskGenService domain.TaskGenerationService,
	maxEventCount int,
	inactivityTimeout time.Duration,
) *UseCase {
	return &UseCase{
		clusterRepo:       clusterRepo,
		eventRepo:         eventRepo,
		taskRepo:          taskRepo,
		taskGenService:    taskGenService,
		maxEventCount:     maxEventCount,
		inactivityTimeout: inactivityTimeout,
	}
}

func (uc *UseCase) ProcessClosableClusters(ctx context.Context, batchSize int) error {
	clusters, err := uc.clusterRepo.FindClosableClusters(
		ctx,
		uc.maxEventCount,
		uc.inactivityTimeout,
		batchSize,
	)
	if err != nil {
		return errors.WrapFail(err, "find closable clusters")
	}

	for _, cluster := range clusters {
		if err := uc.processCluster(ctx, cluster); err != nil {
			return errors.WrapFailf(
				err,
				"process cluster %s",
				errors.Token("cluster_id", cluster.ID.String()),
			)
		}
	}

	return nil
}

func (uc *UseCase) processCluster(ctx context.Context, cluster domain.Cluster) error {
	if err := uc.clusterRepo.UpdateClusterStatus(ctx, cluster.ID, domain.ClusterStatusProcessing); err != nil {
		return errors.WrapFailf(
			err,
			"update cluster status to processing %s",
			errors.Token("cluster_id", cluster.ID.String()),
		)
	}

	events, err := uc.eventRepo.GetEventsByClusterID(ctx, cluster.ID)
	if err != nil {
		return errors.WrapFailf(
			err,
			"get events for cluster %s",
			errors.Token("cluster_id", cluster.ID.String()),
		)
	}

	if len(events) == 0 {
		if err := uc.clusterRepo.UpdateClusterStatus(ctx, cluster.ID, domain.ClusterStatusClosed); err != nil {
			return errors.WrapFailf(
				err,
				"update empty cluster status to closed %s",
				errors.Token("cluster_id", cluster.ID.String()),
			)
		}
		return nil
	}

	generatedTask, err := uc.taskGenService.GenerateTask(ctx, events)
	if err != nil {
		if statusErr := uc.clusterRepo.UpdateClusterStatus(ctx, cluster.ID, domain.ClusterStatusOpen); statusErr != nil {
			return errors.WrapFailf(
				statusErr,
				"rollback cluster status after generation failure %s",
				errors.Token("cluster_id", cluster.ID.String()),
			)
		}
		return errors.WrapFailf(
			err,
			"generate task for cluster %s",
			errors.Token("cluster_id", cluster.ID.String()),
		)
	}

	now := time.Now()
	task := domain.Task{
		ID:          domain.TaskID(uuid.New().String()),
		UserID:      cluster.UserID,
		ClusterID:   cluster.ID,
		Title:       generatedTask.Title,
		Description: generatedTask.Description,
		Duration:    time.Duration(generatedTask.DurationMinutes) * time.Minute,
		Priority:    generatedTask.Priority,
		Deadline:    generatedTask.Deadline,
		StartTime:   generatedTask.StartTime,
		Status:      domain.TaskStatusPending,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := uc.taskRepo.CreateTask(ctx, task); err != nil {
		if statusErr := uc.clusterRepo.UpdateClusterStatus(ctx, cluster.ID, domain.ClusterStatusOpen); statusErr != nil {
			return errors.WrapFailf(
				statusErr,
				"rollback cluster status after task creation failure %s",
				errors.Token("cluster_id", cluster.ID.String()),
			)
		}
		return errors.WrapFailf(
			err,
			"create task for cluster %s",
			errors.Token("cluster_id", cluster.ID.String()),
		)
	}

	if err := uc.clusterRepo.UpdateClusterStatus(ctx, cluster.ID, domain.ClusterStatusClosed); err != nil {
		return errors.WrapFailf(
			err,
			"update cluster status to closed %s",
			errors.Token("cluster_id", cluster.ID.String()),
		)
	}

	return nil
}
