package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	app "github.com/bensuskins/family-hub/internal/application/recipes"
	"github.com/go-chi/chi/v5"
)

type recipeReloadFailureStore struct{ reads int }

func (s *recipeReloadFailureStore) FindByID(context.Context, string) (app.SavedRecipe, error) {
	s.reads++
	if s.reads > 1 {
		return app.SavedRecipe{}, errors.New("reload failed")
	}
	return app.SavedRecipe{ID: "recipe", Title: "Before"}, nil
}
func (*recipeReloadFailureStore) FindAll(context.Context) ([]app.SavedRecipe, error) { return nil, nil }
func (*recipeReloadFailureStore) Create(context.Context, app.SavedRecipe) (app.SavedRecipe, error) {
	return app.SavedRecipe{}, nil
}
func (*recipeReloadFailureStore) Update(context.Context, app.SavedRecipe) error { return nil }
func (*recipeReloadFailureStore) Delete(context.Context, string) error          { return nil }
func TestUpdateRecipeReloadFailureKeepsLegacyIngredientArrays(t *testing.T) {
	store := &recipeReloadFailureStore{}
	handler := NewAPIHandler(nil, nil, nil, nil, nil, nil, nil, app.NewCatalogue(store, nil, nil), nil, nil, nil, nil, nil, "", "")
	router := chi.NewRouter()
	router.Put("/api/recipes/{id}", handler.UpdateRecipe)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPut, "/api/recipes/recipe", strings.NewReader(`{"title":"After","ingredients":[{"name":"Main"}]}`)))
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var body struct {
		Ingredients []struct {
			Items []string `json:"items"`
		} `json:"Ingredients"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Ingredients) != 1 || body.Ingredients[0].Items == nil {
		t.Fatalf("body=%s", response.Body.String())
	}
}
