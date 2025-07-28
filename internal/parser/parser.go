package parser

import (
    "encoding/csv"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "os"
    "path/filepath"
)

type Message struct {
    From    string `json:"from"`
    Date    string `json:"date"`
    Content string `json:"content"`
}

// ParseCSV парсит CSV и возвращает массив строк с данными
func ParseCSV(path string) ([]string, error) {
    file, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    reader := csv.NewReader(file)
    records, err := reader.ReadAll()
    if err != nil {
        return nil, err
    }

    var docs []string
    for _, record := range records {
        docs = append(docs, fmt.Sprintf("%s %s %s", record[0], record[1], record[2]))
    }
    return docs, nil
}

// ParseJSONConversations парсит JSON переписку (массив сообщений)
func ParseJSONConversations(path string) ([]string, error) {
    data, err := ioutil.ReadFile(path)
    if err != nil {
        return nil, err
    }
    var messages []Message
    err = json.Unmarshal(data, &messages)
    if err != nil {
        return nil, err
    }

    var docs []string
    for _, msg := range messages {
        doc := fmt.Sprintf("Сообщение от %s, Дата: %s, Текст: %s", msg.From, msg.Date, msg.Content)
        docs = append(docs, doc)
    }
    return docs, nil
}

// ParseTextFiles читает все .txt файлы из папки и возвращает их содержимое
func ParseTextFiles(dir string) ([]string, error) {
    var docs []string
    files, err := ioutil.ReadDir(dir)
    if err != nil {
        return nil, err
    }
    for _, file := range files {
        if filepath.Ext(file.Name()) == ".txt" {
            data, err := ioutil.ReadFile(filepath.Join(dir, file.Name()))
            if err != nil {
                continue
            }
            docs = append(docs, string(data))
        }
    }
    return docs, nil
}
