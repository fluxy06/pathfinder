package embedding

import (
	"context"
	"fmt"

	"github.com/google/generative-ai-go/genai"
)

// GenerateEmbeddings создаёт эмбеддинги для каждого текста.
func GenerateEmbeddings(ctx context.Context, client *genai.Client, texts []string) ([][]float32, error) {
	model := client.EmbeddingModel("text-embedding-004")

	var embeddings [][]float32
	for _, txt := range texts {
		resp, err := model.EmbedContent(ctx, genai.Text(txt))
		if err != nil {
			return nil, fmt.Errorf("ошибка генерации эмбеддинга: %w", err)
		}

		if len(resp.Embedding.Values) == 0 {
			return nil, fmt.Errorf("пустой эмбеддинг для текста: %s", txt)
		}

		embeddings = append(embeddings, resp.Embedding.Values)
	}
	return embeddings, nil
}
