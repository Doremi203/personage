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
	minSimilarity float64,
	topK int,
	clock func() time.Time,
) *UseCase {
	return &UseCase{
		txProvider:    txProvider,
		logger:        logger,
		embedder:      embedder,
		eventsRepo:    eventsRepo,
		clusterRepo:   clusterRepo,
		minSimilarity: minSimilarity,
		topK:          topK,
		clock:         clock,
	}
}

type UseCase struct {
	embedder      domain.EmbeddingService
	eventsRepo    domain.EventRepo
	clusterRepo   domain.ClusterRepo
	minSimilarity float64
	topK          int

	txProvider tx.Provider
	logger     log.Logger
	clock      func() time.Time
}

func (uc *UseCase) ProcessEvent(ctx context.Context, e domain.Event) error {
	uc.logger.Infof("processing event with id %s", errors.Token("id", e.ID.String()))
	embeddings, err := uc.embedder.GenerateEmbeddings(ctx, []string{string(e.Context)})
	if err != nil {
		return errors.WrapFailf(
			err,
			"generate embedding for event %s",
			errors.Token("id", e.ID.String()),
		)
	}
	if len(embeddings) == 0 {
		return errors.Errorf(
			"generated embeddings are empty for event %s %s",
			errors.Token("id", e.ID.String()),
			errors.Token("context", e.Context),
		)
	}

	embedding := embeddings[0]
	eventWithEmbedding := domain.EventWithEmbedding{
		Event:     e,
		Embedding: embedding,
	}

	similarClusters, err := uc.clusterRepo.FindSimilarClusters(ctx, e.UserID, embedding, uc.topK)
	if err != nil {
		return errors.WrapFailf(
			err,
			"find similar clusters for event %s",
			errors.Token("id", e.ID.String()),
		)
	}

	if len(similarClusters) > 0 {
		uc.logger.Infof(
			"highest similarity for event %s is %s",
			errors.Token("id", e.ID.String()),
			errors.Token("similarity", similarClusters[0].Similarity),
		)
	}
	var chosenCluster domain.Cluster
	if len(similarClusters) > 0 && similarClusters[0].Similarity >= uc.minSimilarity {
		chosenCluster = similarClusters[0].Cluster
		chosenCluster.AddEvent(eventWithEmbedding, uc.clock())
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

	err = uc.txProvider.RunWithTx(ctx, tx.IsolationReadCommitted, func(ctx context.Context) error {
		if err := uc.clusterRepo.UpsertCluster(ctx, chosenCluster); err != nil {
			return errors.WrapFailf(
				err,
				"upsert cluster with %s",
				errors.Token("id", chosenCluster.ID.String()),
			)
		}
		if err := uc.eventsRepo.UpsertEvent(ctx, eventWithEmbedding); err != nil {
			return errors.WrapFailf(
				err,
				"upsert event %s",
				errors.Token("id", eventWithEmbedding.ID.String()),
			)
		}

		return nil
	})
	if err != nil {
		return errors.WrapFailf(
			err,
			"run process event tx",
		)
	}

	uc.logger.Infof("event %s processed", errors.Token("id", e.ID.String()))
	return nil
}
