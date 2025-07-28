package rag

import (
    "context"
    "fmt"
    "strings"

    "pathfinder/internal/chroma"
    "pathfinder/internal/embedding"

    genai "github.com/google/generative-ai-go/genai"
)

func AnswerQuestion(ctx context.Context, client *genai.Client, chromaClient *chroma.ChromaClient, question string) (string, error) {
    // Генерация эмбеддинга вопроса
    embeddingVec, err := embedding.GenerateEmbedding(ctx, client, question)
    if err != nil {
        return "", err
    }

    // Поиск похожих документов в ChromaDB
    docs, metas, err := chromaClient.Query(ctx, embeddingVec, 5)
    if err != nil {
        return "", err
    }

    var contextTexts []string
    var sources []string
    for i, meta := range metas {
        if s, ok := meta["source"]; ok {
            sources = append(sources, s)
        }
        if i < len(docs) {
            contextTexts = append(contextTexts, docs[i])
        }
    }

    prompt := fmt.Sprintf("Ответь на вопрос: %s\nИспользуй контекст:\n%s", question, strings.Join(contextTexts, "\n"))

    // Генерация ответа от модели
    model := client.GenerativeModel("gemini-1.5-flash")
    resp, err := model.GenerateContent(ctx, genai.Text(prompt))
    if err != nil {
        return "", err
    }

    answer := ""
    if len(resp.Candidates) > 0 && len(resp.Candidates[0].Content.Parts) > 0 {
        if text, ok := resp.Candidates[0].Content.Parts[0].(genai.Text); ok {
            answer = string(text)
        }
    }

    return fmt.Sprintf("%s\n\nИсточники:\n- %s", answer, strings.Join(sources, "\n- ")), nil
}
