package services

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type recipeTestTransport func(*http.Request) (*http.Response, error)

func (transport recipeTestTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport(request)
}

func TestRecipeExtractorUsesInjectedClient(t *testing.T) {
	var requested string
	client := &http.Client{Transport: recipeTestTransport(func(request *http.Request) (*http.Response, error) {
		requested = request.URL.String()
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`<html><script type="application/ld+json">{"@type":"Recipe","name":"Soup"}</script></html>`)), Header: make(http.Header), Request: request}, nil
	})}
	extractor := NewRecipeExtractor(client)
	page, _, err := extractor.Load(context.Background(), "https://example.com/recipe")
	if err != nil {
		t.Fatal(err)
	}
	recipe, found := page.Structured()
	if requested != "https://example.com/recipe" || !found || recipe.Title != "Soup" {
		t.Fatalf("request %q, recipe %+v, found %v", requested, recipe, found)
	}
	if err := extractor.Validate("http://127.0.0.1/recipe"); err == nil {
		t.Fatal("injected client disabled URL validation")
	}
}

func TestRecipeExtractorDefaultClientIsSafe(t *testing.T) {
	extractor := NewRecipeExtractor(nil)
	if extractor.client == nil || extractor.client.Timeout != 15*time.Second || extractor.client.CheckRedirect == nil {
		t.Fatal("expected safe client with 15 second timeout and redirect guard")
	}
	if err := extractor.Validate("http://127.0.0.1/recipe"); err == nil {
		t.Fatal("private URL was accepted")
	}
	redirect, err := http.NewRequest(http.MethodGet, "http://127.0.0.1/recipe", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := extractor.client.CheckRedirect(redirect, nil); err == nil {
		t.Fatal("redirect to private URL was accepted")
	}
}
