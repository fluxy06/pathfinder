package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"pathfinder/internal/chroma"
	"pathfinder/internal/config"
	"pathfinder/internal/embedding"
	"pathfinder/internal/utils"
	"github.com/joho/godotenv"
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

func main() {
	// Загружаем .env файл (если есть)
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found or error loading .env")
	}

	ctx := context.Background()

	// Загружаем конфигурацию
	cfg := config.LoadConfig()

	// Инициализируем клиент Google AI
	client, err := genai.NewClient(ctx, option.WithAPIKey(os.Getenv("GOOGLE_API_KEY")))
	if err != nil {
		log.Fatalf("Ошибка создания genai клиента: %v", err)
	}
	defer client.Close()

	// Инициализируем ChromaClient
	chromaClient := chroma.NewChromaClient(cfg.ChromaHost, cfg.ChromaCollection)

	// Пример текста
	text := "Это пример текста, который будет разбит на чанки, преобразован в эмбеддинги и отправлен в ChromaDB."

	// Разбиваем на чанки
	chunks := utils.ChunkText(text, 500, 50)
	fmt.Printf("Разбито на %d чанков\n", len(chunks))

	// Генерируем эмбеддинги
	embeddings, err := embedding.GenerateEmbeddings(ctx, client, chunks)
	if err != nil {
		log.Fatalf("Ошибка генерации эмбеддингов: %v", err)
	}

	// Подготавливаем ID и метаданные
	var ids []string
	var metadatas []map[string]string
	for i := range chunks {
		ids = append(ids, fmt.Sprintf("doc-%d", i))
		metadatas = append(metadatas, map[string]string{
			"chunk_index": fmt.Sprintf("%d", i),
		})
	}
    fmt.Println("Chroma Host:", cfg.ChromaHost)
    fmt.Println("Chroma Collection:", cfg.ChromaCollection)
	// Добавляем документы в Chroma
	err = chromaClient.AddDocuments(ctx, ids, chunks, embeddings, metadatas)
	if err != nil {
		log.Fatalf("Ошибка добавления документов в Chroma: %v", err)
	}

	fmt.Println("Документы успешно добавлены в Chroma.")
}
