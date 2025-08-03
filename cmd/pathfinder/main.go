package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/google/generative-ai-go/genai"
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

	// Получаем переменные окружения
	host := os.Getenv("CHROMA_HOST")
	collection := os.Getenv("CHROMA_COLLECTION")
	apiKey := os.Getenv("GOOGLE_API_KEY")

	if host == "" || collection == "" || apiKey == "" {
		log.Fatal("CHROMA_HOST, CHROMA_COLLECTION и GOOGLE_API_KEY обязательны")
	}

	// Ждем пока сервис Chroma станет доступен
	if err := waitForChromaReady(host+"/health", 30*time.Second); err != nil {
		log.Fatal("Chroma сервис не доступен:", err)
	}

	// Инициализация клиентов
	chromaClient := chroma.NewChromaClient(host, collection)

	genaiClient, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		log.Fatal("Ошибка создания Gemini клиента:", err)
	}
	defer genaiClient.Close()

	// Убедимся, что коллекция существует или создадим её
	if err := chromaClient.EnsureCollection(ctx); err != nil {
		log.Fatal("Ошибка создания коллекции:", err)
	}

	// Путь к папке data внутри контейнера
	dataDir := filepath.Join("data")
	allDocs := []string{}

	// Парсим документы
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

	// Обрабатываем документы и отправляем в Chroma
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

// waitForChromaReady ждёт, пока Chroma API не ответит с HTTP 200 или таймаут
func waitForChromaReady(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		if time.Now().After(deadline) {
			return errors.New("таймаут ожидания сервиса Chroma")
		}
		resp, err := http.Get(url)
		if err == nil && resp.StatusCode == http.StatusOK {
			return nil
		}
		time.Sleep(1 * time.Second)
	}
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

	vector := make([]float32, len(resp.Embedding.Values))
	for i, val := range resp.Embedding.Values {
		vector[i] = float32(val)
	}

	return vector, nil
}
