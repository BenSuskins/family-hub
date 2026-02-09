package config

import (
	"fmt"
	"os"
)

type Config struct {
	DatabasePath    string
	OIDCIssuer      string
	OIDCClientID    string
	OIDCClientSecret string
	OIDCRedirectURL string
	SessionSecret   string
	HAAPIToken      string
	LogLevel        string
	Port            string
}

func Load() (Config, error) {
	config := Config{
		DatabasePath:    envOrDefault("DATABASE_PATH", "./data/family-hub.db"),
		OIDCIssuer:      os.Getenv("OIDC_ISSUER"),
		OIDCClientID:    os.Getenv("OIDC_CLIENT_ID"),
		OIDCClientSecret: os.Getenv("OIDC_CLIENT_SECRET"),
		OIDCRedirectURL: os.Getenv("OIDC_REDIRECT_URL"),
		SessionSecret:   os.Getenv("SESSION_SECRET"),
		HAAPIToken:      os.Getenv("HA_API_TOKEN"),
		LogLevel:        envOrDefault("LOG_LEVEL", "info"),
		Port:            envOrDefault("PORT", "8080"),
	}

	if config.SessionSecret == "" {
		return Config{}, fmt.Errorf("SESSION_SECRET is required")
	}

	return config, nil
}

func envOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
