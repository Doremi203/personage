package taskgeneration

import (
	"context"
	"fmt"
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
	embedder domain.EmbeddingService,
	txProvider tx.Provider,
	logger log.Logger,
	settings domain.GenerationSettingsProvider,
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
		embedder:                    embedder,
		txProvider:                  txProvider,
		logger:                      logger,
		settings:                    settings,
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
	embedder                    domain.EmbeddingService
	txProvider                  tx.Provider
	logger                      log.Logger
	settings                    domain.GenerationSettingsProvider
	clock                       func() time.Time
}

func (uc *UseCase) ProcessClosableClusters(ctx context.Context) error {
	cfg, err := uc.settings.GenerationSettings(ctx)
	if err != nil {
		return errors.WrapFail(err, "load generation settings")
	}

	recovered, err := uc.clusterRepo.RecoverStaleClusters(ctx, cfg.InactivityTimeout)
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
			cfg.MaxEventCount,
			cfg.InactivityTimeout,
			cfg.BatchSize,
		)
		if err != nil {
			return errors.WrapFail(err, "find closable clusters")
		}

		for _, cluster := range clusters {
			if err = uc.clusterRepo.UpdateClusterStatus(txCtx, cluster.ID, domain.ClusterStatusProcessing); err != nil {
				return errors.WrapFailf(
					err,
					"update cluster status to processing for cluster %s for user %s",
					errors.Token("cluster_id", cluster.ID.String()),
					errors.Token("user_id", cluster.UserID.String()),
				)
			}
		}

		return nil
	})
	if err != nil {
		return errors.WrapFail(err, "select and lock closable clusters")
	}

	if len(clusters) > 0 {
		uc.logger.Infof(
			"locked closable clusters for processing %s",
			errors.Token("count", len(clusters)),
		)
	}

	for _, cluster := range clusters {
		if err = uc.processCluster(ctx, cluster, cfg.TaskDuplicateThreshold); err != nil {
			uc.logger.Error(errors.WrapFailf(
				err,
				"process cluster %s for user %s (skipping)",
				errors.Token("cluster_id", cluster.ID.String()),
				errors.Token("user_id", cluster.UserID.String()),
			))
		}
	}

	return nil
}

func (uc *UseCase) processCluster(ctx context.Context, cluster domain.Cluster, duplicateThreshold float64) error {
	uc.logger.Infof(
		"processing cluster %s for user %s",
		errors.Token("cluster_id", cluster.ID.String()),
		errors.Token("user_id", cluster.UserID.String()),
	)

	events, err := uc.eventRepo.GetEventsByClusterID(ctx, cluster.ID)
	if err != nil {
		return errors.WrapFailf(
			err,
			"get events for cluster %s for user %s",
			errors.Token("cluster_id", cluster.ID.String()),
			errors.Token("user_id", cluster.UserID.String()),
		)
	}

	if len(events) == 0 {
		uc.logger.Infof(
			"finalizing empty cluster %s for user %s",
			errors.Token("cluster_id", cluster.ID.String()),
			errors.Token("user_id", cluster.UserID.String()),
		)
		if err := uc.clusterRepo.FinalizeCluster(ctx, cluster.ID, domain.ClusterGenerationOutcomeEmpty, nil); err != nil {
			return errors.WrapFailf(
				err,
				"finalize empty cluster %s for user %s",
				errors.Token("cluster_id", cluster.ID.String()),
				errors.Token("user_id", cluster.UserID.String()),
			)
		}
		return nil
	}

	uc.logger.Infof(
		"classifying actionability for cluster %s for user %s with %s events",
		errors.Token("cluster_id", cluster.ID.String()),
		errors.Token("user_id", cluster.UserID.String()),
		errors.Token("event_count", len(events)),
	)

	profile, err := uc.userProfileService.GetUserProfile(ctx, cluster.UserID)
	if err != nil {
		uc.logger.Warn(errors.WrapFailf(
			err,
			"fetch user profile for cluster %s for user %s",
			errors.Token("cluster_id", cluster.ID.String()),
			errors.Token("user_id", cluster.UserID.String()),
		))
		profile = domain.UserProfile{}
	}

	generationDecision, err := uc.clusterClassificatorService.GetTaskGenerationDecision(ctx, events, profile)
	if err != nil {
		return errors.WrapFailf(
			err,
			"classify cluster actionability for cluster %s for user %s",
			errors.Token("cluster_id", cluster.ID.String()),
			errors.Token("user_id", cluster.UserID.String()),
		)
	}

	if !generationDecision.ShouldGenerate {
		reason := ""
		if generationDecision.Reason != nil {
			reason = *generationDecision.Reason
		}
		uc.logger.Infof(
			"cluster %s for user %s classified as non-actionable: %s",
			errors.Token("cluster_id", cluster.ID.String()),
			errors.Token("user_id", cluster.UserID.String()),
			errors.Token("reason", reason),
		)
		if err := uc.clusterRepo.FinalizeCluster(
			ctx,
			cluster.ID,
			domain.ClusterGenerationOutcomeNonActionable,
			generationDecision.Reason,
		); err != nil {
			return errors.WrapFailf(
				err,
				"finalize non-actionable cluster %s for user %s",
				errors.Token("cluster_id", cluster.ID.String()),
				errors.Token("user_id", cluster.UserID.String()),
			)
		}
		return nil
	}

	uc.logger.Infof(
		"cluster %s for user %s classified as actionable, generating task",
		errors.Token("cluster_id", cluster.ID.String()),
		errors.Token("user_id", cluster.UserID.String()),
	)

	generatedTask, err := uc.taskGenService.GenerateTask(ctx, events, profile)
	if err != nil {
		return errors.WrapFailf(
			err,
			"generate task for cluster %s for user %s",
			errors.Token("cluster_id", cluster.ID.String()),
			errors.Token("user_id", cluster.UserID.String()),
		)
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
		ID:          domain.TaskID(uuid.New().String()),
		UserID:      cluster.UserID,
		ClusterID:   new(cluster.ID),
		Title:       generatedTask.Title,
		Description: generatedTask.Description,
		Duration:    time.Duration(generatedTask.DurationMinutes) * time.Minute,
		Priority:    generatedTask.Priority,
		Deadline:    generatedTask.Deadline,
		StartTime:   generatedTask.StartTime,
		Date:        generatedTask.Date,
		Status:      domain.TaskStatusUnplanned,
		Category:    domain.NewTaskCategory(generatedTask.Category),
		IsApproved:  !requiresModeration,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	uc.logger.Infof(
		"generated task %s for user %s from cluster %s (approved=%s)",
		errors.Token("task_id", task.ID.String()),
		errors.Token("user_id", cluster.UserID.String()),
		errors.Token("cluster_id", cluster.ID.String()),
		errors.Token("is_approved", task.IsApproved),
	)

	embedding := uc.embedTask(ctx, cluster, task)
	if embedding != nil {
		similarTaskID, similarity, found, err := uc.taskRepo.FindMostSimilarActiveTask(ctx, cluster.UserID, embedding)
		if err != nil {
			return errors.WrapFailf(
				err,
				"find most similar active task for user %s for cluster %s",
				errors.Token("user_id", cluster.UserID.String()),
				errors.Token("cluster_id", cluster.ID.String()),
			)
		}

		if found && similarity >= duplicateThreshold {
			reason := fmt.Sprintf("duplicate of task %s (similarity=%.4f)", similarTaskID.String(), similarity)
			uc.logger.Infof(
				"cluster %s for user %s generated a duplicate of task %s (similarity=%s), skipping create",
				errors.Token("cluster_id", cluster.ID.String()),
				errors.Token("user_id", cluster.UserID.String()),
				errors.Token("task_id", similarTaskID.String()),
				errors.Token("similarity", similarity),
			)
			if err := uc.clusterRepo.FinalizeCluster(ctx, cluster.ID, domain.ClusterGenerationOutcomeDuplicate, &reason); err != nil {
				return errors.WrapFailf(
					err,
					"finalize duplicate cluster %s for user %s",
					errors.Token("cluster_id", cluster.ID.String()),
					errors.Token("user_id", cluster.UserID.String()),
				)
			}
			return nil
		}

		task.Embedding = embedding
	}

	err = uc.txProvider.RunWithTx(ctx, tx.IsolationReadCommitted, func(txCtx context.Context) error {
		if err = uc.taskRepo.CreateTask(txCtx, task); err != nil {
			return errors.WrapFailf(
				err,
				"create task %s for user %s for cluster %s",
				errors.Token("task_id", task.ID.String()),
				errors.Token("user_id", cluster.UserID.String()),
				errors.Token("cluster_id", cluster.ID.String()),
			)
		}

		if err = uc.clusterRepo.FinalizeCluster(txCtx, cluster.ID, domain.ClusterGenerationOutcomeTaskGenerated, generationDecision.Reason); err != nil {
			return errors.WrapFailf(
				err,
				"finalize cluster %s for user %s with generated task %s",
				errors.Token("cluster_id", cluster.ID.String()),
				errors.Token("user_id", cluster.UserID.String()),
				errors.Token("task_id", task.ID.String()),
			)
		}

		return nil
	})
	if err != nil {
		return err
	}

	uc.logger.Infof(
		"successfully processed cluster %s for user %s into task %s",
		errors.Token("cluster_id", cluster.ID.String()),
		errors.Token("user_id", cluster.UserID.String()),
		errors.Token("task_id", task.ID.String()),
	)

	return nil
}

// embedTask returns the embedding of the task's content, or nil when embedding
// generation fails or yields no vector. A nil result means the task is created
// without dedup rather than lost.
func (uc *UseCase) embedTask(ctx context.Context, cluster domain.Cluster, task domain.Task) []float32 {
	text := task.Title + "\n" + task.Description

	embeddings, err := uc.embedder.GenerateEmbeddings(ctx, []string{text})
	if err != nil {
		uc.logger.Warn(errors.WrapFailf(
			err,
			"embed generated task %s for user %s for cluster %s (proceeding without dedup)",
			errors.Token("task_id", task.ID.String()),
			errors.Token("user_id", cluster.UserID.String()),
			errors.Token("cluster_id", cluster.ID.String()),
		))
		return nil
	}

	if len(embeddings) == 0 {
		uc.logger.Warn(errors.WrapFailf(
			errors.Error("empty embeddings"),
			"embed generated task %s for user %s for cluster %s (proceeding without dedup)",
			errors.Token("task_id", task.ID.String()),
			errors.Token("user_id", cluster.UserID.String()),
			errors.Token("cluster_id", cluster.ID.String()),
		))
		return nil
	}

	return embeddings[0]
}
