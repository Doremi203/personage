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
	minSimilarity float64,
	closedSimilarityThreshold float64,
	topK int,
	clock func() time.Time,
) *UseCase {
	return &UseCase{
		txProvider:                txProvider,
		logger:                    logger,
		embedder:                  embedder,
		eventsRepo:                eventsRepo,
		clusterRepo:               clusterRepo,
		pauseRepo:                 pauseRepo,
		minSimilarity:             minSimilarity,
		closedSimilarityThreshold: closedSimilarityThreshold,
		topK:                      topK,
		clock:                     clock,
	}
}

type UseCase struct {
	embedder                  domain.EmbeddingService
	eventsRepo                domain.EventRepo
	clusterRepo               domain.ClusterRepo
	pauseRepo                 domain.ProcessingPauseRepo
	minSimilarity             float64
	closedSimilarityThreshold float64
	topK                      int

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

	closedClusters, err := uc.clusterRepo.FindSimilarClosedClusters(ctx, e.UserID, embedding, uc.topK)
	if err != nil {
		return errors.WrapFailf(
			err,
			"find similar closed clusters for event %s for user %s",
			errors.Token("event_id", e.ID.String()),
			errors.Token("user_id", e.UserID.String()),
		)
	}
	if len(closedClusters) > 0 && closedClusters[0].Similarity >= uc.closedSimilarityThreshold {
		uc.logger.Infof(
			"skip event %s for user %s as near-duplicate of closed cluster %s with similarity %s",
			errors.Token("event_id", e.ID.String()),
			errors.Token("user_id", e.UserID.String()),
			errors.Token("cluster_id", closedClusters[0].ID.String()),
			errors.Token("similarity", closedClusters[0].Similarity),
		)
		return nil
	}

	similarClusters, err := uc.clusterRepo.FindSimilarClusters(ctx, e.UserID, embedding, uc.topK)
	if err != nil {
		return errors.WrapFailf(
			err,
			"find similar clusters for event %s for user %s",
			errors.Token("event_id", e.ID.String()),
			errors.Token("user_id", e.UserID.String()),
		)
	}

	if len(similarClusters) > 0 {
		uc.logger.Infof(
			"highest similarity for event %s for user %s is %s",
			errors.Token("event_id", e.ID.String()),
			errors.Token("user_id", e.UserID.String()),
			errors.Token("similarity", similarClusters[0].Similarity),
		)
	}
	var chosenCluster domain.Cluster
	var attachedToExisting bool
	if len(similarClusters) > 0 && similarClusters[0].Similarity >= uc.minSimilarity {
		chosenCluster = similarClusters[0].Cluster
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
