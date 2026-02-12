package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/bensuskins/family-hub/internal/middleware"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/templates/pages"
	"github.com/go-chi/chi/v5"
)

type RecipeHandler struct {
	recipeRepo   repository.RecipeRepository
	categoryRepo repository.CategoryRepository
	mealPlanRepo repository.MealPlanRepository
}

func NewRecipeHandler(recipeRepo repository.RecipeRepository, categoryRepo repository.CategoryRepository, mealPlanRepo repository.MealPlanRepository) *RecipeHandler {
	return &RecipeHandler{recipeRepo: recipeRepo, categoryRepo: categoryRepo, mealPlanRepo: mealPlanRepo}
}

func (handler *RecipeHandler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUser(ctx)

	recipes, err := handler.recipeRepo.FindAll(ctx)
	if err != nil {
		slog.Error("finding recipes", "error", err)
		http.Error(w, "Error loading recipes", http.StatusInternalServerError)
		return
	}

	categories, err := handler.categoryRepo.FindAll(ctx)
	if err != nil {
		slog.Error("finding categories", "error", err)
	}

	categoryMap := make(map[string]string, len(categories))
	for _, c := range categories {
		categoryMap[c.ID] = c.Name
	}

	component := pages.RecipeList(pages.RecipeListProps{
		User:        user,
		Recipes:     recipes,
		CategoryMap: categoryMap,
	})
	component.Render(ctx, w)
}

func (handler *RecipeHandler) Detail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUser(ctx)
	recipeID := chi.URLParam(r, "id")

	recipe, err := handler.recipeRepo.FindByID(ctx, recipeID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	var categoryName string
	if recipe.CategoryID != nil {
		category, err := handler.categoryRepo.FindByID(ctx, *recipe.CategoryID)
		if err == nil {
			categoryName = category.Name
		}
	}

	component := pages.RecipeDetail(pages.RecipeDetailProps{
		User:         user,
		Recipe:       recipe,
		CategoryName: categoryName,
	})
	component.Render(ctx, w)
}

func (handler *RecipeHandler) CreateForm(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUser(ctx)

	categories, err := handler.categoryRepo.FindAll(ctx)
	if err != nil {
		slog.Error("finding categories", "error", err)
	}

	component := pages.RecipeForm(pages.RecipeFormProps{
		User:       user,
		Categories: categories,
		IsEdit:     false,
	})
	component.Render(ctx, w)
}

func (handler *RecipeHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUser(ctx)

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	recipe := models.Recipe{
		Title:           r.FormValue("title"),
		Instructions:    r.FormValue("instructions"),
		Ingredients:     parseIngredientGroups(r),
		CreatedByUserID: user.ID,
	}

	if categoryID := r.FormValue("category_id"); categoryID != "" {
		recipe.CategoryID = &categoryID
	}
	if servingsStr := r.FormValue("servings"); servingsStr != "" {
		if servings, err := strconv.Atoi(servingsStr); err == nil {
			recipe.Servings = &servings
		}
	}
	if prepTime := r.FormValue("prep_time"); prepTime != "" {
		recipe.PrepTime = &prepTime
	}
	if cookTime := r.FormValue("cook_time"); cookTime != "" {
		recipe.CookTime = &cookTime
	}
	if sourceURL := r.FormValue("source_url"); sourceURL != "" {
		recipe.SourceURL = &sourceURL
	}

	if _, err := handler.recipeRepo.Create(ctx, recipe); err != nil {
		slog.Error("creating recipe", "error", err)
		http.Error(w, "Error creating recipe", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/recipes", http.StatusFound)
}

func (handler *RecipeHandler) EditForm(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUser(ctx)
	recipeID := chi.URLParam(r, "id")

	recipe, err := handler.recipeRepo.FindByID(ctx, recipeID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	categories, err := handler.categoryRepo.FindAll(ctx)
	if err != nil {
		slog.Error("finding categories", "error", err)
	}

	component := pages.RecipeForm(pages.RecipeFormProps{
		User:       user,
		Recipe:     &recipe,
		Categories: categories,
		IsEdit:     true,
	})
	component.Render(ctx, w)
}

func (handler *RecipeHandler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	recipeID := chi.URLParam(r, "id")

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	recipe, err := handler.recipeRepo.FindByID(ctx, recipeID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	recipe.Title = r.FormValue("title")
	recipe.Instructions = r.FormValue("instructions")
	recipe.Ingredients = parseIngredientGroups(r)

	if categoryID := r.FormValue("category_id"); categoryID != "" {
		recipe.CategoryID = &categoryID
	} else {
		recipe.CategoryID = nil
	}
	if servingsStr := r.FormValue("servings"); servingsStr != "" {
		if servings, err := strconv.Atoi(servingsStr); err == nil {
			recipe.Servings = &servings
		}
	} else {
		recipe.Servings = nil
	}
	if prepTime := r.FormValue("prep_time"); prepTime != "" {
		recipe.PrepTime = &prepTime
	} else {
		recipe.PrepTime = nil
	}
	if cookTime := r.FormValue("cook_time"); cookTime != "" {
		recipe.CookTime = &cookTime
	} else {
		recipe.CookTime = nil
	}
	if sourceURL := r.FormValue("source_url"); sourceURL != "" {
		recipe.SourceURL = &sourceURL
	} else {
		recipe.SourceURL = nil
	}

	if err := handler.recipeRepo.Update(ctx, recipe); err != nil {
		slog.Error("updating recipe", "error", err)
		http.Error(w, "Error updating recipe", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/recipes/%s", recipeID), http.StatusFound)
}

func (handler *RecipeHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	recipeID := chi.URLParam(r, "id")

	if err := handler.mealPlanRepo.ClearRecipeID(ctx, recipeID); err != nil {
		slog.Error("clearing recipe from meal plans", "error", err)
	}

	if err := handler.recipeRepo.Delete(ctx, recipeID); err != nil {
		slog.Error("deleting recipe", "error", err)
		http.Error(w, "Error deleting recipe", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/recipes", http.StatusFound)
}

func (handler *RecipeHandler) IngredientGroup(w http.ResponseWriter, r *http.Request) {
	indexStr := r.URL.Query().Get("index")
	index, err := strconv.Atoi(indexStr)
	if err != nil {
		index = 0
	}

	component := pages.IngredientGroupFields(index, models.IngredientGroup{})
	component.Render(r.Context(), w)
}

func parseIngredientGroups(r *http.Request) []models.IngredientGroup {
	var groups []models.IngredientGroup
	for i := 0; ; i++ {
		name := r.FormValue(fmt.Sprintf("group_name_%d", i))
		if name == "" && r.FormValue(fmt.Sprintf("group_items_%d", i)) == "" {
			break
		}
		itemsRaw := r.FormValue(fmt.Sprintf("group_items_%d", i))
		var items []string
		for _, line := range strings.Split(itemsRaw, "\n") {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				items = append(items, trimmed)
			}
		}
		if name == "" {
			name = "Main"
		}
		groups = append(groups, models.IngredientGroup{Name: name, Items: items})
	}
	return groups
}
