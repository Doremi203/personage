package clusterization

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
	logger log.Logger,
	txProvider tx.Provider,
	embedder domain.EmbeddingService,
	eventsRepo domain.EventRepo,
	clusterRepo domain.ClusterRepo,
	pauseRepo domain.ProcessingPauseRepo,
	settings domain.GenerationSettingsProvider,
	clock func() time.Time,
) *UseCase {
	return &UseCase{
		txProvider:  txProvider,
		logger:      logger,
		embedder:    embedder,
		eventsRepo:  eventsRepo,
		clusterRepo: clusterRepo,
		pauseRepo:   pauseRepo,
		settings:    settings,
		clock:       clock,
	}
}

type UseCase struct {
	embedder    domain.EmbeddingService
	eventsRepo  domain.EventRepo
	clusterRepo domain.ClusterRepo
	pauseRepo   domain.ProcessingPauseRepo
	settings    domain.GenerationSettingsProvider

	txProvider tx.Provider
	logger     log.Logger
	clock      func() time.Time
}

func (uc *UseCase) ProcessEvent(ctx context.Context, e domain.Event) error {
	uc.logger.Infof(
		"processing event %s for user %s",
		errors.Token("event_id", e.ID.String()),
		errors.Token("user_id", e.UserID.String()),
	)

	paused, err := uc.pauseRepo.IsPaused(ctx, e.UserID)
	if err != nil {
		return errors.WrapFailf(
			err,
			"check processing pause for user %s",
			errors.Token("user_id", e.UserID.String()),
		)
	}
	if paused {
		return errors.WrapFailf(
			domain.ErrProcessingPaused,
			"skip event %s",
			errors.Token("event_id", e.ID.String()),
			errors.Token("user_id", e.UserID.String()),
		)
	}

	embeddings, err := uc.embedder.GenerateEmbeddings(ctx, []string{string(e.Context)})
	if err != nil {
		return errors.WrapFailf(
			err,
			"generate embedding for event %s for user %s",
			errors.Token("event_id", e.ID.String()),
			errors.Token("user_id", e.UserID.String()),
		)
	}
	if len(embeddings) == 0 {
		uc.logger.Infof(
			"generated embeddings are empty for event %s for user %s %s",
			errors.Token("event_id", e.ID.String()),
			errors.Token("user_id", e.UserID.String()),
			errors.Token("context", e.Context),
		)
		return nil
	}

	embedding := embeddings[0]
	eventWithEmbedding := domain.EventWithEmbedding{
		Event:     e,
		Embedding: embedding,
	}

	cfg, err := uc.settings.GenerationSettings(ctx)
	if err != nil {
		return errors.WrapFailf(
			err,
			"load generation settings for event %s for user %s",
			errors.Token("event_id", e.ID.String()),
			errors.Token("user_id", e.UserID.String()),
		)
	}

	closedClusters, err := uc.clusterRepo.FindSimilarClosedClusters(ctx, e.UserID, embedding, cfg.TopK)
	if err != nil {
		return errors.WrapFailf(
			err,
			"find similar closed clusters for event %s for user %s",
			errors.Token("event_id", e.ID.String()),
			errors.Token("user_id", e.UserID.String()),
		)
	}
	if len(closedClusters) > 0 && closedClusters[0].Similarity >= cfg.ClosedSimilarityThreshold {
		uc.logger.Infof(
			"skip event %s for user %s as near-duplicate of closed cluster %s with similarity %s",
			errors.Token("event_id", e.ID.String()),
			errors.Token("user_id", e.UserID.String()),
			errors.Token("cluster_id", closedClusters[0].ID.String()),
			errors.Token("similarity", closedClusters[0].Similarity),
		)
		return nil
	}

	similarClusters, err := uc.clusterRepo.FindSimilarClusters(ctx, e.UserID, embedding, cfg.TopK)
	if err != nil {
		return errors.WrapFailf(
			err,
			"find similar clusters for event %s for user %s",
			errors.Token("event_id", e.ID.String()),
			errors.Token("user_id", e.UserID.String()),
		)
	}

	var bestCandidate domain.ClusterWithSimilarity
	var bestMaxSimilarity float64
	var hasBestCandidate bool
	if len(similarClusters) > 0 {
		clusterIDs := make([]domain.ClusterID, 0, len(similarClusters))
		for _, c := range similarClusters {
			clusterIDs = append(clusterIDs, c.ID)
		}

		maxSimilarities, err := uc.eventsRepo.MaxSimilarityByClusters(ctx, clusterIDs, embedding)
		if err != nil {
			return errors.WrapFailf(
				err,
				"find max member similarity for event %s for user %s",
				errors.Token("event_id", e.ID.String()),
				errors.Token("user_id", e.UserID.String()),
			)
		}

		for _, c := range similarClusters {
			similarity, ok := maxSimilarities[c.ID]
			if !ok {
				similarity = c.Similarity
			}
			if !hasBestCandidate || similarity > bestMaxSimilarity {
				bestCandidate = c
				bestMaxSimilarity = similarity
				hasBestCandidate = true
			}
		}

		uc.logger.Infof(
			"highest max-member similarity for event %s for user %s is %s",
			errors.Token("event_id", e.ID.String()),
			errors.Token("user_id", e.UserID.String()),
			errors.Token("similarity", bestMaxSimilarity),
		)
	}

	var chosenCluster domain.Cluster
	var attachedToExisting bool
	if hasBestCandidate && bestMaxSimilarity >= cfg.MinSimilarity {
		chosenCluster = bestCandidate.Cluster
		eventWithEmbedding.Similarity = bestMaxSimilarity
		chosenCluster.AddEvent(eventWithEmbedding, uc.clock())
		attachedToExisting = true
	} else {
		clusterID := domain.ClusterID(uuid.New().String())
		now := uc.clock()
		chosenCluster = domain.Cluster{
			ID:         clusterID,
			UserID:     e.UserID,
			Centroid:   embedding,
			EventCount: 1,
			Status:     domain.ClusterStatusOpen,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		eventWithEmbedding.Similarity = 1.0
	}
	eventWithEmbedding.ClusterID = chosenCluster.ID

	if attachedToExisting {
		uc.logger.Infof(
			"attached event %s for user %s to existing cluster %s (event_count=%s)",
			errors.Token("event_id", e.ID.String()),
			errors.Token("user_id", e.UserID.String()),
			errors.Token("cluster_id", chosenCluster.ID.String()),
			errors.Token("event_count", chosenCluster.EventCount),
		)
	} else {
		uc.logger.Infof(
			"created new cluster %s for user %s from event %s",
			errors.Token("cluster_id", chosenCluster.ID.String()),
			errors.Token("user_id", e.UserID.String()),
			errors.Token("event_id", e.ID.String()),
		)
	}

	err = uc.txProvider.RunWithTx(ctx, tx.IsolationReadCommitted, func(ctx context.Context) error {
		if err := uc.clusterRepo.UpsertCluster(ctx, chosenCluster); err != nil {
			return errors.WrapFailf(
				err,
				"upsert cluster %s for user %s",
				errors.Token("cluster_id", chosenCluster.ID.String()),
				errors.Token("user_id", e.UserID.String()),
			)
		}
		if err := uc.eventsRepo.UpsertEvent(ctx, eventWithEmbedding); err != nil {
			return errors.WrapFailf(
				err,
				"upsert event %s for user %s",
				errors.Token("event_id", eventWithEmbedding.ID.String()),
				errors.Token("user_id", e.UserID.String()),
			)
		}

		return nil
	})
	if err != nil {
		return errors.WrapFailf(
			err,
			"run process event tx for user %s",
			errors.Token("user_id", e.UserID.String()),
		)
	}

	uc.logger.Infof(
		"event %s processed for user %s into cluster %s",
		errors.Token("event_id", e.ID.String()),
		errors.Token("user_id", e.UserID.String()),
		errors.Token("cluster_id", chosenCluster.ID.String()),
	)
	return nil
}
