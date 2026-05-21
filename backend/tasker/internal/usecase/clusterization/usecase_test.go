package clusterization_test

import (
	"context"
	"testing"
	"time"

	"github.com/Doremi203/personage/backend/libs/go/log"
	"github.com/Doremi203/personage/backend/libs/go/tx"
	"github.com/Doremi203/personage/backend/tasker/internal/domain"
	mock_domain "github.com/Doremi203/personage/backend/tasker/internal/domain/mock"
	"github.com/Doremi203/personage/backend/tasker/internal/usecase/clusterization"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestUseCase_ProcessEvent(t *testing.T) {
	type mocks struct {
		embedder    *mock_domain.MockEmbeddingService
		eventsRepo  *mock_domain.MockEventRepo
		clusterRepo *mock_domain.MockClusterRepo
		txProvider  *stubTxProvider
	}
	type args struct {
		event domain.Event
	}

	const (
		minSimilarity = 0.8
		topK          = 5
	)

	fixedNow := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	clock := func() time.Time { return fixedNow }

	embedding := []float32{0.1, 0.2, 0.3}

	defaultEvent := domain.Event{
		ID:      domain.EventID("e1"),
		UserID:  domain.UserID("u1"),
		Context: domain.NormalizedEventContext("text"),
	}

	tests := []struct {
		name    string
		args    args
		setup   func(m mocks, a args)
		assert  func(t *testing.T, m mocks)
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "embedder error",
			args: args{event: defaultEvent},
			setup: func(m mocks, a args) {
				m.embedder.EXPECT().
					GenerateEmbeddings(gomock.Any(), []string{string(a.event.Context)}).
					Return(nil, assert.AnError)
			},
			wantErr: require.Error,
		},
		{
			name: "empty embeddings skip",
			args: args{event: defaultEvent},
			setup: func(m mocks, a args) {
				m.embedder.EXPECT().
					GenerateEmbeddings(gomock.Any(), []string{string(a.event.Context)}).
					Return([][]float32{}, nil)
			},
			wantErr: require.NoError,
		},
		{
			name: "find similar clusters error",
			args: args{event: defaultEvent},
			setup: func(m mocks, a args) {
				m.embedder.EXPECT().
					GenerateEmbeddings(gomock.Any(), []string{string(a.event.Context)}).
					Return([][]float32{embedding}, nil)
				m.clusterRepo.EXPECT().
					FindSimilarClusters(gomock.Any(), a.event.UserID, embedding, topK).
					Return(nil, assert.AnError)
			},
			wantErr: require.Error,
		},
		{
			name: "similar cluster above threshold reuses cluster",
			args: args{event: defaultEvent},
			setup: func(m mocks, a args) {
				existing := domain.Cluster{
					ID:         domain.ClusterID("existing"),
					UserID:     a.event.UserID,
					Centroid:   []float32{0.1, 0.2, 0.3},
					EventCount: 2,
					Status:     domain.ClusterStatusOpen,
					CreatedAt:  fixedNow.Add(-time.Hour),
					UpdatedAt:  fixedNow.Add(-time.Hour),
				}
				m.embedder.EXPECT().
					GenerateEmbeddings(gomock.Any(), []string{string(a.event.Context)}).
					Return([][]float32{embedding}, nil)
				m.clusterRepo.EXPECT().
					FindSimilarClusters(gomock.Any(), a.event.UserID, embedding, topK).
					Return([]domain.ClusterWithSimilarity{
						{Cluster: existing, Similarity: 0.9},
					}, nil)
				m.clusterRepo.EXPECT().
					UpsertCluster(gomock.Any(), gomock.AssignableToTypeOf(domain.Cluster{})).
					DoAndReturn(func(_ context.Context, c domain.Cluster) error {
						assert.Equal(t, domain.ClusterID("existing"), c.ID)
						assert.Equal(t, 3, c.EventCount)
						assert.Equal(t, domain.ClusterStatusOpen, c.Status)
						assert.Equal(t, fixedNow, c.UpdatedAt)
						return nil
					})
				m.eventsRepo.EXPECT().
					UpsertEvent(gomock.Any(), gomock.AssignableToTypeOf(domain.EventWithEmbedding{})).
					DoAndReturn(func(_ context.Context, e domain.EventWithEmbedding) error {
						assert.Equal(t, domain.ClusterID("existing"), e.ClusterID)
						return nil
					})
			},
			wantErr: require.NoError,
		},
		{
			name: "similar cluster below threshold creates new cluster",
			args: args{event: defaultEvent},
			setup: func(m mocks, a args) {
				existing := domain.Cluster{
					ID:         domain.ClusterID("existing"),
					UserID:     a.event.UserID,
					Centroid:   []float32{0.4, 0.5, 0.6},
					EventCount: 2,
					Status:     domain.ClusterStatusOpen,
					CreatedAt:  fixedNow.Add(-time.Hour),
					UpdatedAt:  fixedNow.Add(-time.Hour),
				}
				m.embedder.EXPECT().
					GenerateEmbeddings(gomock.Any(), []string{string(a.event.Context)}).
					Return([][]float32{embedding}, nil)
				m.clusterRepo.EXPECT().
					FindSimilarClusters(gomock.Any(), a.event.UserID, embedding, topK).
					Return([]domain.ClusterWithSimilarity{
						{Cluster: existing, Similarity: 0.5},
					}, nil)
				m.clusterRepo.EXPECT().
					UpsertCluster(gomock.Any(), gomock.AssignableToTypeOf(domain.Cluster{})).
					DoAndReturn(func(_ context.Context, c domain.Cluster) error {
						assert.NotEqual(t, domain.ClusterID("existing"), c.ID)
						assert.NotEmpty(t, c.ID)
						assert.Equal(t, a.event.UserID, c.UserID)
						assert.Equal(t, 1, c.EventCount)
						assert.Equal(t, domain.ClusterStatusOpen, c.Status)
						assert.Equal(t, embedding, c.Centroid)
						assert.Equal(t, fixedNow, c.CreatedAt)
						assert.Equal(t, fixedNow, c.UpdatedAt)
						return nil
					})
				m.eventsRepo.EXPECT().
					UpsertEvent(gomock.Any(), gomock.AssignableToTypeOf(domain.EventWithEmbedding{})).
					Return(nil)
			},
			wantErr: require.NoError,
		},
		{
			name: "no similar clusters creates new cluster",
			args: args{event: defaultEvent},
			setup: func(m mocks, a args) {
				m.embedder.EXPECT().
					GenerateEmbeddings(gomock.Any(), []string{string(a.event.Context)}).
					Return([][]float32{embedding}, nil)
				m.clusterRepo.EXPECT().
					FindSimilarClusters(gomock.Any(), a.event.UserID, embedding, topK).
					Return(nil, nil)
				m.clusterRepo.EXPECT().
					UpsertCluster(gomock.Any(), gomock.AssignableToTypeOf(domain.Cluster{})).
					DoAndReturn(func(_ context.Context, c domain.Cluster) error {
						assert.NotEmpty(t, c.ID)
						assert.Equal(t, 1, c.EventCount)
						assert.Equal(t, domain.ClusterStatusOpen, c.Status)
						assert.Equal(t, embedding, c.Centroid)
						assert.Equal(t, fixedNow, c.CreatedAt)
						assert.Equal(t, fixedNow, c.UpdatedAt)
						return nil
					})
				m.eventsRepo.EXPECT().
					UpsertEvent(gomock.Any(), gomock.AssignableToTypeOf(domain.EventWithEmbedding{})).
					Return(nil)
			},
			wantErr: require.NoError,
		},
		{
			name: "upsert cluster error",
			args: args{event: defaultEvent},
			setup: func(m mocks, a args) {
				m.embedder.EXPECT().
					GenerateEmbeddings(gomock.Any(), []string{string(a.event.Context)}).
					Return([][]float32{embedding}, nil)
				m.clusterRepo.EXPECT().
					FindSimilarClusters(gomock.Any(), a.event.UserID, embedding, topK).
					Return(nil, nil)
				m.clusterRepo.EXPECT().
					UpsertCluster(gomock.Any(), gomock.AssignableToTypeOf(domain.Cluster{})).
					Return(assert.AnError)
			},
			wantErr: require.Error,
		},
		{
			name: "upsert event error",
			args: args{event: defaultEvent},
			setup: func(m mocks, a args) {
				m.embedder.EXPECT().
					GenerateEmbeddings(gomock.Any(), []string{string(a.event.Context)}).
					Return([][]float32{embedding}, nil)
				m.clusterRepo.EXPECT().
					FindSimilarClusters(gomock.Any(), a.event.UserID, embedding, topK).
					Return(nil, nil)
				m.clusterRepo.EXPECT().
					UpsertCluster(gomock.Any(), gomock.AssignableToTypeOf(domain.Cluster{})).
					Return(nil)
				m.eventsRepo.EXPECT().
					UpsertEvent(gomock.Any(), gomock.AssignableToTypeOf(domain.EventWithEmbedding{})).
					Return(assert.AnError)
			},
			wantErr: require.Error,
		},
		{
			name: "tx provider error skips callback",
			args: args{event: defaultEvent},
			setup: func(m mocks, a args) {
				m.embedder.EXPECT().
					GenerateEmbeddings(gomock.Any(), []string{string(a.event.Context)}).
					Return([][]float32{embedding}, nil)
				m.clusterRepo.EXPECT().
					FindSimilarClusters(gomock.Any(), a.event.UserID, embedding, topK).
					Return(nil, nil)
				m.txProvider.err = assert.AnError
			},
			wantErr: require.Error,
		},
		{
			name: "happy path captures cluster id on event",
			args: args{event: defaultEvent},
			setup: func(m mocks, a args) {
				var capturedClusterID domain.ClusterID
				m.embedder.EXPECT().
					GenerateEmbeddings(gomock.Any(), []string{string(a.event.Context)}).
					Return([][]float32{embedding}, nil)
				m.clusterRepo.EXPECT().
					FindSimilarClusters(gomock.Any(), a.event.UserID, embedding, topK).
					Return(nil, nil)
				m.clusterRepo.EXPECT().
					UpsertCluster(gomock.Any(), gomock.AssignableToTypeOf(domain.Cluster{})).
					DoAndReturn(func(_ context.Context, c domain.Cluster) error {
						capturedClusterID = c.ID
						return nil
					})
				m.eventsRepo.EXPECT().
					UpsertEvent(gomock.Any(), gomock.AssignableToTypeOf(domain.EventWithEmbedding{})).
					DoAndReturn(func(_ context.Context, e domain.EventWithEmbedding) error {
						assert.NotEmpty(t, capturedClusterID)
						assert.Equal(t, capturedClusterID, e.ClusterID)
						assert.Equal(t, embedding, e.Embedding)
						assert.Equal(t, a.event.ID, e.ID)
						return nil
					})
			},
			wantErr: require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			m := mocks{
				embedder:    mock_domain.NewMockEmbeddingService(ctrl),
				eventsRepo:  mock_domain.NewMockEventRepo(ctrl),
				clusterRepo: mock_domain.NewMockClusterRepo(ctrl),
				txProvider:  &stubTxProvider{},
			}
			tt.setup(m, tt.args)

			uc := clusterization.NewUseCase(
				log.Stub{},
				m.txProvider,
				m.embedder,
				m.eventsRepo,
				m.clusterRepo,
				minSimilarity,
				topK,
				clock,
			)

			err := uc.ProcessEvent(t.Context(), tt.args.event)
			tt.wantErr(t, err)
			if tt.assert != nil {
				tt.assert(t, m)
			}
		})
	}
}

type stubTxProvider struct{ err error }

func (s *stubTxProvider) RunWithTx(ctx context.Context, _ tx.Isolation, op func(context.Context) error) error {
	if s.err != nil {
		return s.err
	}
	return op(ctx)
}
