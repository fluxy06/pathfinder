package utils

import "strings"

// ChunkText разбивает текст на куски заданного размера с перекрытием.
func ChunkText(text string, chunkSize, overlap int) []string {
	var chunks []string
	words := strings.Fields(text)

	for i := 0; i < len(words); i += (chunkSize - overlap) {
		end := i + chunkSize
		if end > len(words) {
			end = len(words)
		}
		chunks = append(chunks, strings.Join(words[i:end], " "))
		if end == len(words) {
			break
		}
	}
	return chunks
}
