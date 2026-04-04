package config

import (
	"fmt"
	"os"
)

type Config struct {
	DatabasePath         string
	OIDCIssuer           string
	OIDCClientID         string
	OIDCClientSecret     string
	OIDCRedirectURL      string
	OIDCUserInfoURL      string
	IOSClientID          string
	SessionSecret        string
	SessionEncryptionKey string

	BaseURL              string
	LogLevel             string
	Port                 string
	DevMode              bool
}

func Load() (Config, error) {
	config := Config{
		DatabasePath:         envOrDefault("DATABASE_PATH", "./data/family-hub.db"),
		OIDCIssuer:           os.Getenv("OIDC_ISSUER"),
		OIDCClientID:         os.Getenv("OIDC_CLIENT_ID"),
		OIDCClientSecret:     os.Getenv("OIDC_CLIENT_SECRET"),
		OIDCRedirectURL:      os.Getenv("OIDC_REDIRECT_URL"),
		OIDCUserInfoURL:      envOrDefault("OIDC_USERINFO_URL", os.Getenv("OIDC_ISSUER")+"/api/oidc/userinfo"),
		IOSClientID:          os.Getenv("IOS_OIDC_CLIENT_ID"),
		SessionSecret:        os.Getenv("SESSION_SECRET"),
		SessionEncryptionKey: os.Getenv("SESSION_ENCRYPTION_KEY"),

		BaseURL:              envOrDefault("BASE_URL", "http://localhost:8080"),
		LogLevel:             envOrDefault("LOG_LEVEL", "info"),
		Port:                 envOrDefault("PORT", "8080"),
		DevMode:              os.Getenv("DEV_MODE") == "true",
	}

	if config.SessionSecret == "" {
		return Config{}, fmt.Errorf("SESSION_SECRET is required")
	}

	if len(config.SessionSecret) < 32 {
		return Config{}, fmt.Errorf("SESSION_SECRET must be at least 32 characters")
	}

	if config.OIDCIssuer == "" && !config.DevMode {
		return Config{}, fmt.Errorf("OIDC_ISSUER is required (set DEV_MODE=true to bypass for local development)")
	}

	return config, nil
}

func envOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
