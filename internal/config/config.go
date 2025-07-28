package config

import "os"

type Config struct {
	GoogleAPIKey     string
	ChromaHost       string
	ChromaCollection string
}

func LoadConfig() *Config {
	return &Config{
		GoogleAPIKey:     os.Getenv("GOOGLE_API_KEY"),
		ChromaHost:       getEnv("CHROMA_HOST", "http://localhost:8000"),
		ChromaCollection: getEnv("CHROMA_COLLECTION", "default_collection"),
	}
}

func getEnv(key, defaultValue string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	}
	return val
}
