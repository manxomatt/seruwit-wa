package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port             string
	APIKey           string
	DBDialect        string
	DBAddress        string
	CRMWebhookURL    string
	CRMWebhookSecret string
	LogLevel         string
}

func Load() *Config {
	_ = godotenv.Load()

	return &Config{
		Port:             getEnv("PORT", "8080"),
		APIKey:           getEnv("API_KEY", "seruwit-secret-key-change-me"),
		DBDialect:        getEnv("DB_DIALECT", "sqlite"),
		DBAddress:        getEnv("DB_ADDRESS", "file:data/seruwit-wa.db?_foreign_keys=on"),
		CRMWebhookURL:    getEnv("CRM_WEBHOOK_URL", ""),
		CRMWebhookSecret: getEnv("CRM_WEBHOOK_SECRET", ""),
		LogLevel:         getEnv("LOG_LEVEL", "INFO"),
	}
}

func getEnv(key, defaultVal string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	return val
}
