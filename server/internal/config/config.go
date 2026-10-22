package config

import (
	"fmt"
	"os"
)

type Config struct {
	APNsKeyID         string
	APNsTeamID        string
	APNsTopic         string
	APNsKeyPath       string
	GoogleMapsAPIKey  string
	DatabasePath      string
	OIDCIssuer        string
	OIDCClientID      string
	OIDCUserInfoURL   string
	HuxleyBaseURL     string
	HuxleyAccessToken string

	LogLevel string
	Port     string
}

func Load() (Config, error) {
	config := Config{
		APNsKeyID:         os.Getenv("APNS_KEY_ID"),
		APNsTeamID:        os.Getenv("APNS_TEAM_ID"),
		APNsTopic:         os.Getenv("APNS_TOPIC"),
		APNsKeyPath:       os.Getenv("APNS_KEY_PATH"),
		GoogleMapsAPIKey:  os.Getenv("GOOGLE_MAPS_API_KEY"),
		DatabasePath:      envOrDefault("DATABASE_PATH", "./data/family-hub.db"),
		OIDCIssuer:        os.Getenv("OIDC_ISSUER"),
		OIDCClientID:      os.Getenv("OIDC_CLIENT_ID"),
		OIDCUserInfoURL:   envOrDefault("OIDC_USERINFO_URL", os.Getenv("OIDC_ISSUER")+"/api/oidc/userinfo"),
		HuxleyBaseURL:     envOrDefault("HUXLEY_BASE_URL", "https://huxley2.azurewebsites.net"),
		HuxleyAccessToken: os.Getenv("HUXLEY_ACCESS_TOKEN"),

		LogLevel: envOrDefault("LOG_LEVEL", "info"),
		Port:     envOrDefault("PORT", "8080"),
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

	if config.OIDCIssuer == "" {
		return Config{}, fmt.Errorf("OIDC_ISSUER is required")
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
