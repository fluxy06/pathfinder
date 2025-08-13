package utils

import "strings"

func ChunkText(text string, chunkSize, overlap int) []string {
	var chunks []string
	words := strings.Fields(text)
	if chunkSize <= 0 {
		return []string{text}
	}
	step := chunkSize - overlap
	if step < 1 { step = 1 }

	for i := 0; i < len(words); i += step {
		end := i + chunkSize
		if end > len(words) { end = len(words) }
		chunks = append(chunks, strings.Join(words[i:end], " "))
		if end == len(words) { break }
	}
	return chunks
}
