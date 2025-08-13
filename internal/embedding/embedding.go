package embedding

import (
	"context"
	"fmt"

	"github.com/google/generative-ai-go/genai"
)

const EmbeddingModel = "gemini-embedding-001" // актуальная на 2025-08

func GenerateEmbedding(ctx context.Context, client *genai.Client, text string, dim int) ([]float32, error) {
	model := client.EmbeddingModel(EmbeddingModel)
	if dim > 0 {
		model = model.WithOutputDimensionality(int32(dim))
	}
	resp, err := model.EmbedContent(ctx, genai.Text(text))
	if err != nil {
		return nil, fmt.Errorf("embedding error: %w", err)
	}
	if resp.Embedding == nil || len(resp.Embedding.Values) == 0 {
		return nil, fmt.Errorf("empty embedding")
	}
	vec := make([]float32, len(resp.Embedding.Values))
	for i, v := range resp.Embedding.Values {
		vec[i] = float32(v)
	}
	return vec, nil
}

// батч-версия — экономит вызовы
func GenerateEmbeddingsBatch(ctx context.Context, client *genai.Client, texts []string, dim int) ([][]float32, error) {
	m := client.EmbeddingModel(EmbeddingModel)
	if dim > 0 {
		m = m.WithOutputDimensionality(int32(dim))
	}
	b := m.NewBatch()
	for _, t := range texts {
		b.AddContent(genai.Text(t))
	}
	res, err := m.BatchEmbedContents(ctx, b)
	if err != nil {
		return nil, fmt.Errorf("batch embedding error: %w", err)
	}
	out := make([][]float32, 0, len(res.Embeddings))
	for _, e := range res.Embeddings {
		vec := make([]float32, len(e.Values))
		for i, v := range e.Values {
			vec[i] = float32(v)
		}
		out = append(out, vec)
	}
	return out, nil
}
