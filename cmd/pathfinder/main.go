package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"pathfinder/internal/chroma"
	"pathfinder/internal/config"
	"pathfinder/internal/rag"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type AskRequest struct {
	Question string `json:"question"`
}

type AskResponse struct {
	Answer string `json:"answer"`
	Error  string `json:"error,omitempty"`
}

func main() {
	// Загружаем конфиг
	cfg := config.LoadConfig()

	// Проверяем обязательные переменные
	if cfg.GoogleAPIKey == "" {
		log.Fatal("GOOGLE_API_KEY environment variable is required")
	}
	if cfg.ChromaHost == "" {
		log.Fatal("CHROMA_HOST environment variable is required")
	}
	if cfg.ChromaCollection == "" {
		log.Fatal("CHROMA_COLLECTION environment variable is required")
	}

	// Инициализируем Google AI клиента
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(cfg.GoogleAPIKey))
	if err != nil {
		log.Fatalf("Ошибка создания genai клиента: %v", err)
	}
	defer client.Close()

	// Инициализируем Chroma
	chromaClient := chroma.NewChromaClient(cfg.ChromaHost, cfg.ChromaCollection)

	// Роут /ask
	http.HandleFunc("/ask", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req AskRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		answer, err := rag.AnswerQuestion(ctx, client, chromaClient, req.Question)
		resp := AskResponse{}
		if err != nil {
			log.Printf("Error answering question: %v", err)
			resp.Error = fmt.Sprintf("Ошибка обработки запроса: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			resp.Answer = answer
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	// Запуск сервера
	port := "8080"
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}
	log.Printf("PathFinder API запущен на порту %s...\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// TODO:
// Понял в чем ошибка!!
//При новом запуске коллекция не сохраняется и соответственно смысла от uuid4, который мы ранее получали и в окружение ставили нет,
//Требуется рефакторинг кода, под автоматическую генерацию uuid4 коллекции!