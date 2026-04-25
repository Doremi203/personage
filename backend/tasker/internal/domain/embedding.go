package domain

import "context"

//go:generate mockgen -source=embedding.go -destination=mock/embedding_mock.go -typed

type EmbeddingService interface {
	GenerateEmbeddings(ctx context.Context, strings []string) ([][]float32, error)
}
