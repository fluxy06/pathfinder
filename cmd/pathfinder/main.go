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

	"github.com/joho/godotenv"
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
	// Загружаем .env
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found or error loading .env")
	}

	// Загружаем конфиг
	cfg := config.LoadConfig()

	// Инициализируем Google AI клиента
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(os.Getenv("GOOGLE_API_KEY")))
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
			resp.Error = fmt.Sprintf("Ошибка: %v", err)
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
