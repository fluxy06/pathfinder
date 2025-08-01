package embedding

import (
	"context"
	"fmt"

	"github.com/google/generative-ai-go/genai"
)

// GenerateEmbedding создает эмбеддинг для одного текста
func GenerateEmbedding(ctx context.Context, client *genai.Client, text string) ([]float32, error) {
	model := client.EmbeddingModel("models/embedding-001")

	resp, err := model.EmbedContent(ctx, genai.Text(text))
	if err != nil {
		return nil, fmt.Errorf("ошибка генерации эмбеддинга: %w", err)
	}

	if resp.Embedding == nil || len(resp.Embedding.Values) == 0 {
		return nil, fmt.Errorf("пустой эмбеддинг")
	}

	// Преобразуем []float64 → []float32
	values := make([]float32, len(resp.Embedding.Values))
	for i, v := range resp.Embedding.Values {
		values[i] = float32(v)
	}

	return values, nil
}

// GenerateEmbeddings создает эмбеддинги для массива текстов
func GenerateEmbeddings(ctx context.Context, client *genai.Client, texts []string) ([][]float32, error) {
	model := client.EmbeddingModel("models/embedding-001")

	var result [][]float32

	for _, text := range texts {
		resp, err := model.EmbedContent(ctx, genai.Text(text))
		if err != nil {
			return nil, fmt.Errorf("ошибка генерации эмбеддинга для текста: %w", err)
		}
		if resp.Embedding == nil || len(resp.Embedding.Values) == 0 {
			return nil, fmt.Errorf("пустой эмбеддинг")
		}

		values := make([]float32, len(resp.Embedding.Values))
		for i, v := range resp.Embedding.Values {
			values[i] = float32(v)
		}
		result = append(result, values)
	}

	return result, nil
}
