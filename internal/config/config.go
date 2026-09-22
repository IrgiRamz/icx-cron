package config

import (
	"bufio"
	"os"
	"strings"
)

type Config struct {
	Port             string
	DBPath           string
	APIKey           string
	MaxWorkers       int
	LogRetentionDays int
	AdminUsername    string
	AdminPassword    string
	SessionSecret    string
}

func loadEnvFile(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		return // .env file is optional
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			val = strings.Trim(val, `"'`)
			if os.Getenv(key) == "" {
				os.Setenv(key, val)
			}
		}
	}
}

func LoadConfig() *Config {
	loadEnvFile(".env")

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

	adminUser := os.Getenv("ADMIN_USERNAME")
	if adminUser == "" {
		adminUser = "admin"
	}

	adminPass := os.Getenv("ADMIN_PASSWORD")
	if adminPass == "" {
		adminPass = "admin"
	}

	sessionSecret := os.Getenv("SESSION_SECRET")
	if sessionSecret == "" {
		sessionSecret = "secret-key-bebas-acak"
	}

	return &Config{
		Port:             port,
		DBPath:           dbPath,
		APIKey:           apiKey,
		MaxWorkers:       30,
		LogRetentionDays: 7,
		AdminUsername:    adminUser,
		AdminPassword:    adminPass,
		SessionSecret:    sessionSecret,
	}
}
