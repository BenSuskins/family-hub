package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	recipeapp "github.com/bensuskins/family-hub/internal/application/recipes"
)

type recipePageFake struct{}

func (recipePageFake) Structured() (recipeapp.Recipe, bool) {
	return recipeapp.Recipe{Title: "Soup", Ingredients: []string{"water"}, Steps: []string{"Cook"}}, true
}
func (recipePageFake) Video() (recipeapp.Recipe, bool) { return recipeapp.Recipe{}, false }
func (recipePageFake) Metadata() recipeapp.Metadata    { return recipeapp.Metadata{} }

type recipeImportFake struct{}

func (recipeImportFake) Validate(string) error                        { return nil }
func (recipeImportFake) Resolve(_ context.Context, url string) string { return url }
func (recipeImportFake) Load(_ context.Context, url string) (recipeapp.Page, string, error) {
	return recipePageFake{}, url, nil
}
func (recipeImportFake) OEmbed(context.Context, string) (recipeapp.Recipe, bool) {
	return recipeapp.Recipe{}, false
}
func (recipeImportFake) CaptionEmbed(context.Context, string) (recipeapp.Recipe, bool) {
	return recipeapp.Recipe{}, false
}
func (recipeImportFake) CleanCaption(value string) string { return value }
func (recipeImportFake) ApplyCaption(recipe recipeapp.Recipe, _ string) recipeapp.Recipe {
	return recipe
}
func (recipeImportFake) ResolveURL(_ string, value string) string { return value }
func TestRecipeImportResponseUsesExistingJSONFields(t *testing.T) {
	provider := recipeImportFake{}
	handler := NewAPIHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, recipeapp.New(provider, provider), "", "", "")
	response := httptest.NewRecorder()
	handler.ExtractRecipe(response, httptest.NewRequest(http.MethodPost, "/api/recipes/extract", strings.NewReader(`{"url":"https://example.com/recipe"}`)))
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body) != 3 || body["title"] != "Soup" || body["ingredients"].([]any)[0] != "water" || body["steps"].([]any)[0] != "Cook" {
		t.Fatalf("body=%s", response.Body.String())
	}
}
