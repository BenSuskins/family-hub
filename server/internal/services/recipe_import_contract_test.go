package services

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	recipeapp "github.com/bensuskins/family-hub/internal/application/recipes"
)

type contractPage struct {
	recipe   recipeapp.Recipe
	complete bool
}

func (page contractPage) Structured() (recipeapp.Recipe, bool) { return page.recipe, page.complete }
func (contractPage) Video() (recipeapp.Recipe, bool)           { return recipeapp.Recipe{}, false }
func (contractPage) Metadata() recipeapp.Metadata              { return recipeapp.Metadata{} }

type contractRecipeSource struct{}

func (contractRecipeSource) Validate(string) error                        { return nil }
func (contractRecipeSource) Resolve(_ context.Context, url string) string { return url }
func (contractRecipeSource) Load(_ context.Context, url string) (recipeapp.Page, string, error) {
	if strings.HasSuffix(url, "/blocked") {
		return nil, url, errors.New("blocked")
	}
	return contractPage{recipe: recipeapp.Recipe{Title: "Cake", Ingredients: []string{"flour"}, Steps: []string{"Mix"}}, complete: true}, url, nil
}
func (contractRecipeSource) OEmbed(_ context.Context, url string) (recipeapp.Recipe, bool) {
	if strings.HasSuffix(url, "/blocked") {
		return recipeapp.Recipe{Title: "Fallback"}, true
	}
	return recipeapp.Recipe{}, false
}
func (contractRecipeSource) CaptionEmbed(context.Context, string) (recipeapp.Recipe, bool) {
	return recipeapp.Recipe{}, false
}
func (contractRecipeSource) CleanCaption(value string) string { return value }
func (contractRecipeSource) ApplyCaption(recipe recipeapp.Recipe, _ string) recipeapp.Recipe {
	return recipe
}
func (contractRecipeSource) ResolveURL(_ string, value string) string { return value }
func TestRecipeImportProviderContract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/recipe":
			writer.Write([]byte(`<script type="application/ld+json">{"@type":"Recipe","name":"Cake","recipeIngredient":["flour"],"recipeInstructions":[{"@type":"HowToStep","text":"Mix"}]}</script>`))
		case "/blocked":
			writer.WriteHeader(http.StatusForbidden)
		case "/oembed":
			writer.Write([]byte(`{"title":"Fallback"}`))
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	real := NewRecipeExtractorForTest(server.Client()).WithOEmbedProvider(hostOf(server.URL), server.URL+"/oembed")
	fake := contractRecipeSource{}
	for name, service := range map[string]*recipeapp.Service{"http": recipeapp.New(real, real), "fake": recipeapp.New(fake, fake)} {
		t.Run(name, func(t *testing.T) {
			recipe, err := service.Import(context.Background(), server.URL+"/recipe")
			if err != nil || recipe.Title != "Cake" || len(recipe.Ingredients) != 1 || recipe.Ingredients[0] != "flour" || len(recipe.Steps) != 1 || recipe.Steps[0] != "Mix" {
				t.Fatalf("structured=%+v err=%v", recipe, err)
			}
			recipe, err = service.Import(context.Background(), server.URL+"/blocked")
			if err != nil || recipe.Title != "Fallback" {
				t.Fatalf("fallback=%+v err=%v", recipe, err)
			}
		})
	}
}
