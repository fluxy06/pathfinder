package rag

import (
	"context"
	"fmt"
	"strings"

	"pathfinder/internal/chroma"
	"pathfinder/internal/embedding"

	genai "github.com/google/generative-ai-go/genai"
)

const GenerationModel = "gemini-2.5-pro"

func AnswerQuestion(ctx context.Context, client *genai.Client, store *chroma.ChromaClient, question string, topK int) (string, error) {
	// embed question
	qvec, err := embedding.GenerateEmbedding(ctx, client, question, 3072)
	if err != nil { return "", err }

	docs, metas, dists, err := store.Query(ctx, qvec, topK)
	if err != nil { return "", err }

	var b strings.Builder
	for i, d := range docs {
		fmt.Fprintf(&b, "### Фрагмент %d\n%s\n\n", i+1, d)
	}

	prompt := fmt.Sprintf(`Ты — помощник, отвечающий строго по контексту. 
Вопрос: %s

Контекст (несколько фрагментов, можно цитировать):
%s

Требования:
- Ответь кратко и по делу, если информации мало — честно скажи об этом.
- В конце выдай раздел "Источники" со списком ссылок/путей из метаданных.
`, question, b.String())

	model := client.GenerativeModel(GenerationModel)
	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil { return "", err }

	answer := ""
	if len(resp.Candidates) > 0 && len(resp.Candidates[0].Content.Parts) > 0 {
		if t, ok := resp.Candidates[0].Content.Parts[0].(genai.Text); ok {
			answer = string(t)
		}
	}

	// источники
	var srcs []string
	for i := range metas {
		if s, ok := metas[i]["source"]; ok && s != "" {
			srcs = append(srcs, fmt.Sprintf("- %s (dist=%.4f)", s, dists[i]))
		}
	}
	if len(srcs) == 0 {
		srcs = append(srcs, "- (метаданные источника недоступны)")
	}

	return fmt.Sprintf("%s\n\nИсточники:\n%s", answer, strings.Join(srcs, "\n")), nil
}
