package taskgeneration

import (
	"context"
	"time"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/Doremi203/personage/backend/libs/go/log"
	"github.com/Doremi203/personage/backend/libs/go/tx"
	"github.com/Doremi203/personage/backend/tasker/internal/domain"
	"github.com/google/uuid"
)

const rollbackTimeout = 5 * time.Second

func NewUseCase(
	clusterRepo domain.ClusterRepo,
	eventRepo domain.EventRepo,
	taskRepo domain.TaskRepo,
	taskGenService domain.TaskGenerationService,
	txProvider tx.Provider,
	logger log.Logger,
	maxEventCount int,
	inactivityTimeout time.Duration,
) *UseCase {
	return &UseCase{
		clusterRepo:       clusterRepo,
		eventRepo:         eventRepo,
		taskRepo:          taskRepo,
		taskGenService:    taskGenService,
		txProvider:        txProvider,
		logger:            logger,
		maxEventCount:     maxEventCount,
		inactivityTimeout: inactivityTimeout,
	}
}

type UseCase struct {
	clusterRepo       domain.ClusterRepo
	eventRepo         domain.EventRepo
	taskRepo          domain.TaskRepo
	taskGenService    domain.TaskGenerationService
	txProvider        tx.Provider
	logger            log.Logger
	maxEventCount     int
	inactivityTimeout time.Duration
}

func (uc *UseCase) ProcessClosableClusters(ctx context.Context, batchSize int) error {
	recovered, err := uc.clusterRepo.RecoverStaleClusters(ctx, uc.inactivityTimeout)
	if err != nil {
		return errors.WrapFail(err, "recover stale processing clusters")
	}
	if recovered > 0 {
		uc.logger.Infof("recovered stale processing clusters: %s",
			errors.Token("count", recovered),
		)
	}

	var clusters []domain.Cluster
	err = uc.txProvider.RunWithTx(ctx, tx.IsolationReadCommitted, func(txCtx context.Context) error {
		clusters, err = uc.clusterRepo.FindClosableClusters(
			txCtx,
			uc.maxEventCount,
			uc.inactivityTimeout,
			batchSize,
		)
		if err != nil {
			return errors.WrapFail(err, "find closable clusters")
		}

		for _, cluster := range clusters {
			if err = uc.clusterRepo.UpdateClusterStatus(txCtx, cluster.ID, domain.ClusterStatusProcessing); err != nil {
				return errors.WrapFailf(
					err,
					"update cluster status to processing %s",
					errors.Token("cluster_id", cluster.ID.String()),
				)
			}
		}

		return nil
	})
	if err != nil {
		return errors.WrapFail(err, "select and lock closable clusters")
	}

	for _, cluster := range clusters {
		if err = uc.processCluster(ctx, cluster); err != nil {
			uc.logger.Error(errors.WrapFailf(
				err,
				"process cluster %s (skipping)",
				errors.Token("cluster_id", cluster.ID.String()),
			))
		}
	}

	return nil
}

func (uc *UseCase) processCluster(ctx context.Context, cluster domain.Cluster) error {
	events, err := uc.eventRepo.GetEventsByClusterID(ctx, cluster.ID)
	if err != nil {

		return errors.Join(
			uc.rollbackClusterStatus(ctx, cluster.ID),
			errors.WrapFail(err, "get events for cluster"),
		)
	}

	if len(events) == 0 {
		if err := uc.clusterRepo.UpdateClusterStatus(ctx, cluster.ID, domain.ClusterStatusClosed); err != nil {
			return errors.WrapFail(err, "update empty cluster status to closed")
		}
		return nil
	}

	generatedTask, err := uc.taskGenService.GenerateTask(ctx, events)
	if err != nil {
		return errors.Join(
			uc.rollbackClusterStatus(ctx, cluster.ID),
			errors.WrapFail(err, "generate task"),
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
		Status:      domain.TaskStatusUnplanned,
		Category:    domain.NewTaskCategory(generatedTask.Category),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	err = uc.txProvider.RunWithTx(ctx, tx.IsolationReadCommitted, func(txCtx context.Context) error {
		if err = uc.taskRepo.CreateTask(txCtx, task); err != nil {
			return errors.WrapFail(err, "create task for cluster")
		}

		if err = uc.clusterRepo.UpdateClusterStatus(txCtx, cluster.ID, domain.ClusterStatusClosed); err != nil {
			return errors.WrapFailf(err, "update cluster status to closed")
		}

		return nil
	})
	if err != nil {
		return errors.Join(uc.rollbackClusterStatus(ctx, cluster.ID), err)
	}

	return nil
}

func (uc *UseCase) rollbackClusterStatus(ctx context.Context, clusterID domain.ClusterID) error {
	rollbackCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), rollbackTimeout)
	defer cancel()

	if err := uc.clusterRepo.UpdateClusterStatus(rollbackCtx, clusterID, domain.ClusterStatusOpen); err != nil {
		return errors.WrapFailf(err, "rollback cluster status to open")
	}

	return nil
}
