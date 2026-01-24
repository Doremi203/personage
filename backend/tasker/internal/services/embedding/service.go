package embedding

import (
	"context"

	"github.com/cloudwego/eino/components/embedding"
)

func NewEinoService(
	embedder embedding.Embedder,
) *einoService {
	return &einoService{
		embedder: embedder,
	}
}

type einoService struct {
	embedder embedding.Embedder
}

func (e *einoService) GenerateEmbeddings(ctx context.Context, strings []string) ([][]float32, error) {
	embeddings, err := e.embedder.EmbedStrings(ctx, strings)
	if err != nil {
		return nil, err
	}

	result := make([][]float32, len(embeddings))
	for i, emb := range embeddings {
		float32Emb := make([]float32, len(emb))
		for j, val := range emb {
			float32Emb[j] = float32(val)
		}
		result[i] = float32Emb
	}

	return result, nil
}
