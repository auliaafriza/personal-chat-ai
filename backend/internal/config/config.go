package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port             string
	Env              string
	DatabaseURL      string
	GroqAPIKey       string
	CORSOrigins      []string
}

func Load() (*Config, error) {
	// Load .env if present (no-op in production)
	_ = godotenv.Load()

	cfg := &Config{
		Port:            getEnv("PORT", "8080"),
		Env:             getEnv("ENV", "development"),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		GroqAPIKey:      os.Getenv("GROQ_API_KEY"),
		CORSOrigins:     parseOrigins(getEnv("CORS_ORIGINS", "http://localhost:3000")),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("missing env: DATABASE_URL")
	}
	if cfg.GroqAPIKey == "" {
		return nil, fmt.Errorf("missing env: GROQ_API_KEY")
	}

	return cfg, nil
}

func (c *Config) IsDev() bool {
	return c.Env == "development"
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parseOrigins(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
