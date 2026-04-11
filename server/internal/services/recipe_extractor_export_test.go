package services

import "net/http"

// NewRecipeExtractorForTest builds a RecipeExtractor wired for use with
// httptest servers: it skips URL validation (so loopback addresses work)
// and uses the supplied HTTP client. This file has the _test.go suffix
// so the symbol is only compiled into test binaries.
func NewRecipeExtractorForTest(client *http.Client) *RecipeExtractor {
	return &RecipeExtractor{
		client:      client,
		validateURL: func(string) error { return nil },
	}
}
