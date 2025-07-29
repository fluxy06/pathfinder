package embedding

import (
	"context"
	"fmt"

	"github.com/google/generative-ai-go/genai"
)

func GenerateEmbeddings(ctx context.Context, client *genai.Client, texts []string) ([][]float32, error) {
	model := client.EmbeddingModel("gemini-embedding-001")

	var embeddings [][]float32
	for _, txt := range texts {
		resp, err := model.EmbedContent(ctx, genai.Text(txt))
		if err != nil {
			return nil, fmt.Errorf("ошибка генерации эмбеддинга: %w", err)
		}
		embeddings = append(embeddings, resp.Embedding.Values)
	}
	return embeddings, nil
}

func GenerateEmbedding(ctx context.Context, client *genai.Client, text string) ([]float32, error) {
	model := client.EmbeddingModel("gemini-embedding-001")
	resp, err := model.EmbedContent(ctx, genai.Text(text))
	if err != nil {
		return nil, fmt.Errorf("ошибка генерации эмбеддинга: %w", err)
	}
	return resp.Embedding.Values, nil
}
