package config

import (
	"strings"
	"testing"
)

func TestLoadCommuteConfiguration(t *testing.T) {
	t.Setenv("SESSION_SECRET", strings.Repeat("s", 32))
	t.Setenv("DEV_MODE", "true")
	t.Setenv("HUXLEY_BASE_URL", "")
	t.Setenv("HUXLEY_ACCESS_TOKEN", "")
	configuration, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if configuration.HuxleyBaseURL != "https://huxley2.azurewebsites.net" || configuration.HuxleyAccessToken != "" {
		t.Fatal("unexpected commute defaults")
	}
	t.Setenv("HUXLEY_BASE_URL", "https://rail.example.com")
	t.Setenv("HUXLEY_ACCESS_TOKEN", "test-token")
	configuration, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	if configuration.HuxleyBaseURL != "https://rail.example.com" || configuration.HuxleyAccessToken != "test-token" {
		t.Fatal("commute environment was not loaded")
	}
}
