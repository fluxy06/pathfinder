package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"

	"pathfinder/internal/chroma"
	"pathfinder/internal/parser"
	"pathfinder/internal/rag"
	"pathfinder/internal/utils"
	"pathfinder/internal/embedding"
)

const (
	defaultChunkSize = 180 // слов
	defaultOverlap   = 30
)

func main() {
	ctx := context.Background()

	// сабкоманды: ingest / ask
	if len(os.Args) < 2 {
		fmt.Println("usage:")
		fmt.Println("  pathfinder ingest --data ./data")
		fmt.Println("  pathfinder ask \"ваш вопрос\" -k 5")
		os.Exit(1)
	}

	host := getEnv("CHROMA_HOST", "http://localhost:8000")
	collection := getEnv("CHROMA_COLLECTION", "pathfinder")
	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		log.Fatal("GOOGLE_API_KEY обязателен")
	}

	// ждём chroma
	if err := waitForChromaReady(host+"/health", 30*time.Second); err != nil {
		log.Fatal("Chroma недоступна:", err)
	}

	chromaClient := chroma.NewChromaClient(host, collection)
	if err := chromaClient.EnsureCollection(ctx); err != nil {
		log.Fatal("Коллекция:", err)
	}

	genaiClient, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		log.Fatal("Gemini client error:", err)
	}
	defer genaiClient.Close()

	switch os.Args[1] {
	case "ingest":
		fs := flag.NewFlagSet("ingest", flag.ExitOnError)
		dataDir := fs.String("data", "./data", "папка с данными")
		chunk := fs.Int("chunk", defaultChunkSize, "размер чанка (в словах)")
		ov := fs.Int("overlap", defaultOverlap, "перекрытие (в словах)")
		dim := fs.Int("dim", 3072, "размерность эмбеддинга (рекомендуется 3072 для gemini-embedding-001)")
		_ = fs.Parse(os.Args[2:])
		if err := ingest(ctx, genaiClient, chromaClient, *dataDir, *chunk, *ov, *dim); err != nil {
			log.Fatal(err)
		}
	case "ask":
		fs := flag.NewFlagSet("ask", flag.ExitOnError)
		k := fs.Int("k", 5, "сколько фрагментов вернуть")
		_ = fs.Parse(os.Args[2:])
		if fs.NArg() < 1 {
			log.Fatal("нужен вопрос: pathfinder ask \"...\"")
		}
		question := strings.Join(fs.Args(), " ")
		ans, err := rag.AnswerQuestion(ctx, genaiClient, chromaClient, question, *k)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(ans)
	default:
		log.Fatalf("неизвестная команда: %s", os.Args[1])
	}
}

func ingest(ctx context.Context, genaiClient *genai.Client, chromaClient *chroma.ChromaClient, dataDir string, chunkSize, overlap, dim int) error {
	var parsed []parser.ParsedDoc

	txt, _ := parser.ParseTextFiles(dataDir)
	parsed = append(parsed, txt...)

	jsonFiles, _ := filepath.Glob(filepath.Join(dataDir, "*.json"))
	for _, p := range jsonFiles {
		docs, _ := parser.ParseJSONConversations(p)
		parsed = append(parsed, docs...)
	}

	csvFiles, _ := filepath.Glob(filepath.Join(dataDir, "*.csv"))
	for _, p := range csvFiles {
		docs, _ := parser.ParseCSV(p)
		parsed = append(parsed, docs...)
	}

	log.Printf("Найдено документов: %d", len(parsed))

	// чанкуем
	var chunks []string
	var metas []map[string]string
	for _, d := range parsed {
		for _, c := range utils.ChunkText(d.Text, chunkSize, overlap) {
			if strings.TrimSpace(c) == "" { continue }
			chunks = append(chunks, c)
			metas = append(metas, d.Meta)
		}
	}
	log.Printf("Чанков к индексации: %d", len(chunks))

	// батчим эмбеддинги
	const batchSize = 64
	idCounter := 0
	for i := 0; i < len(chunks); i += batchSize {
		j := i + batchSize
		if j > len(chunks) { j = len(chunks) }
		batch := chunks[i:j]
		vecs, err := embedding.GenerateEmbeddingsBatch(ctx, genaiClient, batch, dim)
		if err != nil {
			return fmt.Errorf("batch %d..%d: %w", i, j, err)
		}
		ids := make([]string, 0, len(batch))
		mds := make([]map[string]string, 0, len(batch))
		for range batch {
			ids = append(ids, fmt.Sprintf("doc-%d", idCounter))
			mds = append(mds, metas[i])
			idCounter++
		}
		if err := chromaClient.Upsert(ctx, ids, batch, vecs, mds); err != nil {
			return fmt.Errorf("upsert: %w", err)
		}
	}

	log.Printf("Готово! Загружено чанков: %d", idCounter)
	return nil
}

// --- utils ---

func waitForChromaReady(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		if time.Now().After(deadline) {
			return errors.New("таймаут ожидания Chroma")
		}
		resp, err := http.Get(url)
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return nil
		}
		time.Sleep(1 * time.Second)
	}
}

func getEnv(k, def string) string {
	if v := os.Getenv(k); v != "" { return v }
	return def
}
