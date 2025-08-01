package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/google/generative-ai-go/genai"
	"github.com/joho/godotenv"
	"google.golang.org/api/option"

	"pathfinder/internal/chroma"
	"pathfinder/internal/parser"
	"pathfinder/internal/utils"
)

const (
	chunkSize = 100
	overlap   = 20
)

func main() {
	ctx := context.Background()

	if err := godotenv.Load("../../.env"); err != nil {
		log.Fatal("Ошибка загрузки .env:", err)
	}

	host := os.Getenv("CHROMA_HOST")
	collection := os.Getenv("CHROMA_COLLECTION")
	apiKey := os.Getenv("GOOGLE_API_KEY")
	if host == "" || collection == "" || apiKey == "" {
		log.Fatal("CHROMA_HOST, CHROMA_COLLECTION и GOOGLE_API_KEY обязательны")
	}

	chromaClient := chroma.NewChromaClient(host, collection)

	genaiClient, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		log.Fatal("Ошибка создания Gemini клиента:", err)
	}
	defer genaiClient.Close()

	if err := chromaClient.EnsureCollection(ctx); err != nil {
		log.Fatal("Ошибка создания коллекции:", err)
	}

	dataDir := filepath.Join("../../data")
	allDocs := []string{}

	txtDocs, _ := parser.ParseTextFiles(dataDir)
	allDocs = append(allDocs, txtDocs...)

	jsonFiles, _ := filepath.Glob(filepath.Join(dataDir, "*.json"))
	for _, jsonPath := range jsonFiles {
		jsonDocs, _ := parser.ParseJSONConversations(jsonPath)
		allDocs = append(allDocs, jsonDocs...)
	}

	csvFiles, _ := filepath.Glob(filepath.Join(dataDir, "*.csv"))
	for _, csvPath := range csvFiles {
		csvDocs, _ := parser.ParseCSV(csvPath)
		allDocs = append(allDocs, csvDocs...)
	}

	log.Printf("Найдено %d документов", len(allDocs))

	idCounter := 0
	for _, doc := range allDocs {
		chunks := utils.ChunkText(doc, chunkSize, overlap)
		for _, chunk := range chunks {
			embedding, err := getEmbedding(ctx, genaiClient, chunk)
			if err != nil {
				log.Println("Ошибка эмбеддинга:", err)
				continue
			}

			id := fmt.Sprintf("doc-%d", idCounter)
			err = chromaClient.AddDocuments(ctx,
				[]string{id},
				[]string{chunk},
				[][]float32{embedding},
				[]map[string]string{{"source": "ingest"}})

			if err != nil {
				log.Println("Ошибка добавления в Chroma:", err)
			} else {
				log.Printf("Добавлено: %s\n", id)
			}
			idCounter++
		}
	}

	log.Printf("Готово! Загружено %d чанков\n", idCounter)
}

func getEmbedding(ctx context.Context, client *genai.Client, text string) ([]float32, error) {
	model := client.EmbeddingModel("models/embedding-001")

	resp, err := model.EmbedContent(ctx, genai.Text(text))
	if err != nil {
		return nil, fmt.Errorf("ошибка получения эмбеддинга: %w", err)
	}

	if resp.Embedding == nil || len(resp.Embedding.Values) == 0 {
		return nil, fmt.Errorf("пустой ответ на эмбеддинг")
	}

	// Преобразуем []float32 из Values
	vector := make([]float32, len(resp.Embedding.Values))
	for i, val := range resp.Embedding.Values {
		vector[i] = float32(val)
	}

	return vector, nil
}