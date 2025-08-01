package chroma

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type ChromaClient struct {
	host       string
	collection string
}

func NewChromaClient(host, collection string) *ChromaClient {
	return &ChromaClient{
		host:       host,
		collection: collection,
	}
}

// EnsureCollection проверяет существование коллекции и создаёт её при необходимости (API v1)
func (c *ChromaClient) EnsureCollection(ctx context.Context) error {
	url := fmt.Sprintf("%s/api/v1/collections/%s", c.host, c.collection)

	// Проверка
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("ошибка создания запроса на проверку коллекции: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("ошибка запроса при проверке коллекции: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fmt.Printf("Коллекция '%s' уже существует\n", c.collection)
		return nil
	}

	if resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ошибка при проверке коллекции: %s", string(body))
	}

	// Создание коллекции
	createURL := fmt.Sprintf("%s/api/v1/collections", c.host)
	payload := map[string]string{
		"name": c.collection,
	}
	body, _ := json.Marshal(payload)

	createReq, err := http.NewRequestWithContext(ctx, http.MethodPost, createURL, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("ошибка создания запроса на создание коллекции: %w", err)
	}
	createReq.Header.Set("Content-Type", "application/json")

	createResp, err := http.DefaultClient.Do(createReq)
	if err != nil {
		return fmt.Errorf("ошибка при создании коллекции: %w", err)
	}
	defer createResp.Body.Close()

	if createResp.StatusCode != http.StatusOK && createResp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(createResp.Body)
		return fmt.Errorf("ошибка создания коллекции: %s", string(respBody))
	}

	fmt.Printf("Коллекция '%s' успешно создана\n", c.collection)
	return nil
}

// AddDocuments добавляет документы в коллекцию (API v1)
func (c *ChromaClient) AddDocuments(ctx context.Context, ids []string, documents []string, embeddings [][]float32, metadatas []map[string]string) error {
	if len(ids) != len(documents) || len(ids) != len(embeddings) {
		return fmt.Errorf("количество ids, documents и embeddings не совпадает")
	}

	var records []map[string]interface{}
	for i := range ids {
		record := map[string]interface{}{
			"id":        ids[i],
			"document":  documents[i],
			"embedding": embeddings[i],
		}
		if i < len(metadatas) && len(metadatas[i]) > 0 {
			record["metadata"] = metadatas[i]
		}
		records = append(records, record)
	}

	body := map[string]interface{}{
		"records": records,
	}

	data, _ := json.MarshalIndent(body, "", "  ")
	fmt.Println("AddDocuments payload:", string(data))

	url := fmt.Sprintf("%s/api/v1/collections/%s/upsert", c.host, c.collection)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("ошибка создания запроса: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("ошибка выполнения запроса к Chroma: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Chroma вернула ошибку: %s, %s", resp.Status, string(respBody))
	}

	return nil
}

// Query выполняет поиск по эмбеддингу (API v1)
func (c *ChromaClient) Query(ctx context.Context, embedding []float32, topK int) ([]string, []map[string]string, error) {
	body := map[string]interface{}{
		"query_embeddings": [][]float32{embedding},
		"n_results":        topK,
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, nil, fmt.Errorf("ошибка маршалинга тела запроса: %w", err)
	}
	url := fmt.Sprintf("%s/api/v1/collections/%s/query", c.host, c.collection)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(data))
	if err != nil {
		return nil, nil, fmt.Errorf("ошибка создания запроса: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("ошибка запроса к Chroma: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, nil, fmt.Errorf("ошибка от Chroma: %s, %s", resp.Status, string(respBody))
	}

	var result struct {
		Records []struct {
			Document string            `json:"document"`
			Metadata map[string]string `json:"metadata"`
		} `json:"records"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, nil, fmt.Errorf("ошибка парсинга ответа Chroma: %w", err)
	}

	var docs []string
	var metas []map[string]string
	for _, r := range result.Records {
		docs = append(docs, r.Document)
		metas = append(metas, r.Metadata)
	}

	return docs, metas, nil
}
