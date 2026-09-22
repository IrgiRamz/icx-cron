package config

import (
	"os"
)

type Config struct {
	Port           string
	DBPath         string
	APIKey         string
	MaxWorkers     int
	LogRetentionDays int
}

func LoadConfig() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "cron.db"
	}

	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		apiKey = "iconix-cron-secret-key"
	}

	return &Config{
		Port:             port,
		DBPath:           dbPath,
		APIKey:           apiKey,
		MaxWorkers:       30,
		LogRetentionDays: 7,
	}
}
