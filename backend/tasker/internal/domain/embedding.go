package domain

import "context"

type EmbeddingService interface {
	GenerateEmbeddings(ctx context.Context, strings []string) ([][]float32, error)
}
