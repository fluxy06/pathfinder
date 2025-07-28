package chroma

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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

func (c *ChromaClient) AddDocuments(ctx context.Context, ids []string, texts []string, embeddings [][]float32, metadatas []map[string]string) error {
	body := map[string]interface{}{
		"ids":        ids,
		"documents":  texts,
		"embeddings": embeddings,
		"metadatas":  metadatas,
	}
	data, _ := json.Marshal(body)

	url := fmt.Sprintf("%s/collections/%s/add", c.host, c.collection)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("ошибка создания запроса: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("ошибка отправки запроса: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ошибка от Chroma: %s", resp.Status)
	}

	return nil
}
