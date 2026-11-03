package config

import "testing"

func TestAPIDocsRequireExplicitOptIn(t *testing.T) {
	t.Setenv("OIDC_ISSUER", "https://identity.example")
	for _, value := range []string{"", "false", "TRUE", "1", "true"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("API_DOCS_ENABLED", value)
			configuration, err := Load()
			if err != nil {
				t.Fatal(err)
			}
			if configuration.APIDocsEnabled != (value == "true") {
				t.Fatalf("API_DOCS_ENABLED=%q: enabled=%v", value, configuration.APIDocsEnabled)
			}
		})
	}
}
