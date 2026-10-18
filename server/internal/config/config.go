package config

import (
	"fmt"
	"os"
)

type Config struct {
	APNsKeyID            string
	APNsTeamID           string
	APNsTopic            string
	APNsKeyPath          string
	GoogleMapsAPIKey     string
	DatabasePath         string
	OIDCIssuer           string
	OIDCClientID         string
	OIDCRedirectURL      string
	OIDCUserInfoURL      string
	SessionSecret        string
	SessionEncryptionKey string
	HuxleyBaseURL        string
	HuxleyAccessToken    string

	BaseURL  string
	LogLevel string
	Port     string
	DevMode  bool
}

func Load() (Config, error) {
	config := Config{
		APNsKeyID:            os.Getenv("APNS_KEY_ID"),
		APNsTeamID:           os.Getenv("APNS_TEAM_ID"),
		APNsTopic:            os.Getenv("APNS_TOPIC"),
		APNsKeyPath:          os.Getenv("APNS_KEY_PATH"),
		GoogleMapsAPIKey:     os.Getenv("GOOGLE_MAPS_API_KEY"),
		DatabasePath:         envOrDefault("DATABASE_PATH", "./data/family-hub.db"),
		OIDCIssuer:           os.Getenv("OIDC_ISSUER"),
		OIDCClientID:         os.Getenv("OIDC_CLIENT_ID"),
		OIDCRedirectURL:      os.Getenv("OIDC_REDIRECT_URL"),
		OIDCUserInfoURL:      envOrDefault("OIDC_USERINFO_URL", os.Getenv("OIDC_ISSUER")+"/api/oidc/userinfo"),
		SessionSecret:        os.Getenv("SESSION_SECRET"),
		SessionEncryptionKey: os.Getenv("SESSION_ENCRYPTION_KEY"),
		HuxleyBaseURL:        envOrDefault("HUXLEY_BASE_URL", "https://huxley2.azurewebsites.net"),
		HuxleyAccessToken:    os.Getenv("HUXLEY_ACCESS_TOKEN"),

		BaseURL:  envOrDefault("BASE_URL", "http://localhost:8080"),
		LogLevel: envOrDefault("LOG_LEVEL", "info"),
		Port:     envOrDefault("PORT", "8080"),
		DevMode:  os.Getenv("DEV_MODE") == "true",
	}

	pushValues := []string{config.APNsKeyID, config.APNsTeamID, config.APNsTopic, config.APNsKeyPath}
	pushCount := 0
	for _, value := range pushValues {
		if value != "" {
			pushCount++
		}
	}
	if pushCount != 0 && pushCount != len(pushValues) {
		return Config{}, fmt.Errorf("APNS_KEY_ID, APNS_TEAM_ID, APNS_TOPIC, and APNS_KEY_PATH must be configured together")
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

func (config Config) PushConfigured() bool {
	return config.APNsKeyID != "" && config.APNsTeamID != "" && config.APNsTopic != "" && config.APNsKeyPath != ""
}
