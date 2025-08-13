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
	return &ChromaClient{host: host, collection: collection}
}

// EnsureCollection: если нет — создаём с cosine
func (c *ChromaClient) EnsureCollection(ctx context.Context) error {
	getURL := fmt.Sprintf("%s/api/v1/collections/%s", c.host, c.collection)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, getURL, nil)
	resp, err := http.DefaultClient.Do(req)
	if err == nil && resp.StatusCode == http.StatusOK {
		resp.Body.Close()
		fmt.Printf("Коллекция '%s' существует\n", c.collection)
		return nil
	}
	if resp != nil { resp.Body.Close() }

	createURL := fmt.Sprintf("%s/api/v1/collections", c.host)
	body := map[string]any{
		"name": c.collection,
		"metadata": map[string]any{
			"hnsw:space": "cosine",
		},
	}
	data, _ := json.Marshal(body)
	cr, err := http.NewRequestWithContext(ctx, http.MethodPost, createURL, bytes.NewBuffer(data))
	if err != nil { return err }
	cr.Header.Set("Content-Type", "application/json")
	rs, err := http.DefaultClient.Do(cr)
	if err != nil { return err }
	defer rs.Body.Close()
	if rs.StatusCode != http.StatusOK && rs.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(rs.Body)
		return fmt.Errorf("create collection failed: %s", string(b))
	}
	fmt.Printf("Коллекция '%s' создана (cosine)\n", c.collection)
	return nil
}

func (c *ChromaClient) Upsert(ctx context.Context, ids []string, documents []string, embeddings [][]float32, metadatas []map[string]string) error {
	if len(ids) != len(documents) || len(ids) != len(embeddings) {
		return fmt.Errorf("len mismatch ids/docs/embeddings")
	}
	records := make([]map[string]any, 0, len(ids))
	for i := range ids {
		rec := map[string]any{
			"id":        ids[i],
			"document":  documents[i],
			"embedding": embeddings[i],
		}
		if i < len(metadatas) && metadatas[i] != nil {
			rec["metadata"] = metadatas[i]
		}
		records = append(records, rec)
	}
	payload := map[string]any{ "records": records }
	data, _ := json.Marshal(payload)

	url := fmt.Sprintf("%s/api/v1/collections/%s/upsert", c.host, c.collection)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil { return err }
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("chroma upsert error: %s %s", resp.Status, string(b))
	}
	return nil
}

// Query по схеме arrays + include
type QueryResult struct {
	IDs        [][]string               `json:"ids"`
	Documents  [][]string               `json:"documents"`
	Metadatas  [][]map[string]string    `json:"metadatas"`
	Distances  [][]float64              `json:"distances"`
}

func (c *ChromaClient) Query(ctx context.Context, embedding []float32, topK int) ([]string, []map[string]string, []float64, error) {
	body := map[string]any{
		"query_embeddings": [][]float32{embedding},
		"n_results":        topK,
		"include":          []string{"documents", "metadatas", "distances", "ids"},
	}
	data, _ := json.Marshal(body)
	url := fmt.Sprintf("%s/api/v1/collections/%s/query", c.host, c.collection)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil { return nil, nil, nil, err }
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, nil, nil, fmt.Errorf("chroma query error: %s %s", resp.Status, string(b))
	}
	var qr QueryResult
	if err := json.NewDecoder(resp.Body).Decode(&qr); err != nil {
		return nil, nil, nil, err
	}
	docs := []string{}
	metas := []map[string]string{}
	dists := []float64{}
	if len(qr.Documents) > 0 {
		docs = append(docs, qr.Documents[0]...)
	}
	if len(qr.Metadatas) > 0 {
		metas = append(metas, qr.Metadatas[0]...)
	}
	if len(qr.Distances) > 0 {
		dists = append(dists, qr.Distances[0]...)
	}
	return docs, metas, dists, nil
}
