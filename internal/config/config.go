package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL           string
	APIKey                string
	SessionSecret         string
	MailgunAPIKey         string
	MailgunDomain         string
	WebhookDestinationURL string
	Port                  string
}

func Load() (*Config, error) {
	// Load .env file if it exists (ignore error if not found)
	_ = godotenv.Load()

	cfg := &Config{
		DatabaseURL:           getEnv("DATABASE_URL", "postgres://localhost:5432/load_calendar?sslmode=disable"),
		APIKey:                getEnv("API_KEY", ""),
		SessionSecret:         getEnv("SESSION_SECRET", "default-secret-change-in-production"),
		MailgunAPIKey:         getEnv("MAILGUN_API_KEY", ""),
		MailgunDomain:         getEnv("MAILGUN_DOMAIN", ""),
		WebhookDestinationURL: getEnv("WEBHOOK_DESTINATION_URL", ""),
		Port:                  getEnv("PORT", "8080"),
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
