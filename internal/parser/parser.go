package parser

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

type Message struct {
	From    string `json:"from"`
	Date    string `json:"date"`
	Content string `json:"content"`
}

type ParsedDoc struct {
	Text string
	Meta map[string]string
}

func ParseCSV(path string) ([]ParsedDoc, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	records, err := r.ReadAll()
	if err != nil {
		return nil, err
	}

	var docs []ParsedDoc
	for i, rec := range records {
		// безопасно склеим первые три колонки, если их меньше — просто возьмём то, что есть
		txt := ""
		for j := 0; j < len(rec) && j < 3; j++ {
			if j > 0 {
				txt += " "
			}
			txt += rec[j]
		}
		docs = append(docs, ParsedDoc{
			Text: txt,
			Meta: map[string]string{
				"source": path,
				"type":   "csv",
				"row":    fmt.Sprintf("%d", i),
			},
		})
	}
	return docs, nil
}

func ParseJSONConversations(path string) ([]ParsedDoc, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var messages []Message
	if err := json.Unmarshal(data, &messages); err != nil {
		return nil, err
	}
	var docs []ParsedDoc
	for i, m := range messages {
		doc := fmt.Sprintf("Сообщение от %s, Дата: %s, Текст: %s", m.From, m.Date, m.Content)
		docs = append(docs, ParsedDoc{
			Text: doc,
			Meta: map[string]string{
				"source": path,
				"type":   "json",
				"idx":    fmt.Sprintf("%d", i),
			},
		})
	}
	return docs, nil
}

func ParseTextFiles(dir string) ([]ParsedDoc, error) {
	var docs []ParsedDoc
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil { return err }
		if d.IsDir() { return nil }
		if filepath.Ext(d.Name()) == ".txt" {
			data, err := os.ReadFile(p)
			if err == nil {
				docs = append(docs, ParsedDoc{
					Text: string(data),
					Meta: map[string]string{
						"source": p,
						"type":   "txt",
					},
				})
			}
		}
		return nil
	})
	return docs, err
}
