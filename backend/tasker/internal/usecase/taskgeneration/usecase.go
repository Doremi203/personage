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

func NewUseCase(
	clusterRepo domain.ClusterRepo,
	eventRepo domain.EventRepo,
	taskRepo domain.TaskRepo,
	moderationRepo domain.ManualModerationRepo,
	actionabilityService domain.ClusterClassificatorService,
	taskGenService domain.TaskGenerationService,
	userProfileService domain.UserProfileService,
	txProvider tx.Provider,
	logger log.Logger,
	maxEventCount int,
	inactivityTimeout time.Duration,
	clock func() time.Time,
) *UseCase {
	return &UseCase{
		clusterRepo:                 clusterRepo,
		eventRepo:                   eventRepo,
		taskRepo:                    taskRepo,
		moderationRepo:              moderationRepo,
		clusterClassificatorService: actionabilityService,
		taskGenService:              taskGenService,
		userProfileService:          userProfileService,
		txProvider:                  txProvider,
		logger:                      logger,
		maxEventCount:               maxEventCount,
		inactivityTimeout:           inactivityTimeout,
		clock:                       clock,
	}
}

type UseCase struct {
	clusterRepo                 domain.ClusterRepo
	eventRepo                   domain.EventRepo
	taskRepo                    domain.TaskRepo
	moderationRepo              domain.ManualModerationRepo
	clusterClassificatorService domain.ClusterClassificatorService
	taskGenService              domain.TaskGenerationService
	userProfileService          domain.UserProfileService
	txProvider                  tx.Provider
	logger                      log.Logger
	maxEventCount               int
	inactivityTimeout           time.Duration
	clock                       func() time.Time
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
		return errors.WrapFail(err, "get events for cluster")
	}

	if len(events) == 0 {
		if err := uc.clusterRepo.FinalizeCluster(ctx, cluster.ID, domain.ClusterGenerationOutcomeEmpty, nil); err != nil {
			return errors.WrapFail(err, "finalize empty cluster")
		}
		return nil
	}

	profile, err := uc.userProfileService.GetUserProfile(ctx, cluster.UserID)
	if err != nil {
		uc.logger.Warn(errors.WrapFailf(
			err,
			"fetch user profile for cluster classification %s",
			errors.Token("user_id", cluster.UserID.String()),
		))
		profile = domain.UserProfile{}
	}

	generationDecision, err := uc.clusterClassificatorService.GetTaskGenerationDecision(ctx, events, profile)
	if err != nil {
		return errors.WrapFail(err, "classify cluster actionability")
	}

	if !generationDecision.ShouldGenerate {
		if err := uc.clusterRepo.FinalizeCluster(
			ctx,
			cluster.ID,
			domain.ClusterGenerationOutcomeNonActionable,
			generationDecision.Reason,
		); err != nil {
			return errors.WrapFail(err, "finalize non-actionable cluster")
		}
		return nil
	}

	generatedTask, err := uc.taskGenService.GenerateTask(ctx, events)
	if err != nil {
		return errors.WrapFail(err, "generate task")
	}

	requiresModeration, err := uc.moderationRepo.RequiresModeration(ctx, cluster.UserID)
	if err != nil {
		return errors.WrapFailf(
			err,
			"check manual moderation for user %s",
			errors.Token("user_id", cluster.UserID.String()),
		)
	}

	now := uc.clock()
	task := domain.Task{
		ID:               domain.TaskID(uuid.New().String()),
		UserID:           cluster.UserID,
		ClusterID:        new(cluster.ID),
		Title:            generatedTask.Title,
		Description:      generatedTask.Description,
		Duration:         time.Duration(generatedTask.DurationMinutes) * time.Minute,
		Priority:         generatedTask.Priority,
		Deadline:         generatedTask.Deadline,
		StartTime:        generatedTask.StartTime,
		Status:           domain.TaskStatusUnplanned,
		Category:         domain.NewTaskCategory(generatedTask.Category),
		EvidenceEventIDs: generatedTask.EvidenceEventIDs,
		IsApproved:       !requiresModeration,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	err = uc.txProvider.RunWithTx(ctx, tx.IsolationReadCommitted, func(txCtx context.Context) error {
		if err = uc.taskRepo.CreateTask(txCtx, task); err != nil {
			return errors.WrapFail(err, "create task for cluster")
		}

		if err = uc.clusterRepo.FinalizeCluster(txCtx, cluster.ID, domain.ClusterGenerationOutcomeTaskGenerated, generationDecision.Reason); err != nil {
			return errors.WrapFail(err, "finalize cluster with generated task")
		}

		return nil
	})
	if err != nil {
		return err
	}

	uc.logger.Infof(
		"successfully processed cluster %s %s",
		errors.Token("cluster_id", cluster.ID.String()),
		errors.Token("task_id", task.ID.String()),
	)

	return nil
}
