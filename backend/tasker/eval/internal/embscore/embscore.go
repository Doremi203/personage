package embscore

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"time"
)

const (
	DefaultBaseURL = "https://openrouter.ai/api/v1"
	DefaultModel   = "openai/text-embedding-3-small"
)

// Scorer calls an OpenAI-compatible embeddings endpoint to compute cosine similarity.
type Scorer struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

// New returns a Scorer. Empty model or baseURL fall back to the OpenRouter defaults.
func New(apiKey, model, baseURL string) *Scorer {
	if model == "" {
		model = DefaultModel
	}
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return &Scorer{
		apiKey:  apiKey,
		model:   model,
		baseURL: baseURL,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

type embRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type embResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
}

// EmbedBatch returns one []float64 embedding per input text, in order.
// Returns nil, nil for an empty input slice.
func (s *Scorer) EmbedBatch(ctx context.Context, texts []string) ([][]float64, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	body, err := json.Marshal(embRequest{Model: s.model, Input: texts})
	if err != nil {
		return nil, fmt.Errorf("marshal embed request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create embed request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embed request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(resp.Body)
		return nil, fmt.Errorf("embed API HTTP %d: %s", resp.StatusCode, buf.String())
	}

	var parsed embResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decode embed response: %w", err)
	}

	result := make([][]float64, len(texts))
	for _, d := range parsed.Data {
		if d.Index >= 0 && d.Index < len(result) {
			result[d.Index] = d.Embedding
		}
	}
	return result, nil
}

// CosineSim returns the cosine similarity of two vectors, clamped to [0, 1].
func CosineSim(a, b []float64) float64 {
	if len(a) == 0 || len(a) != len(b) {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	sim := dot / (math.Sqrt(normA) * math.Sqrt(normB))
	if sim < 0 {
		return 0
	}
	return sim
}
