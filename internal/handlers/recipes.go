package handlers

import (
	"encoding/base64"
	"fmt"
	"io"
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
		Steps:           parseSteps(r),
		Ingredients:     parseIngredientGroups(r),
		MealType:        parseMealType(r.FormValue("meal_type")),
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

	if len(recipe.Steps) == 0 && recipe.Instructions != "" {
		recipe.Steps = splitIntoSteps(recipe.Instructions)
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
	recipe.Steps = parseSteps(r)
	recipe.Ingredients = parseIngredientGroups(r)
	recipe.MealType = parseMealType(r.FormValue("meal_type"))
	// Instructions intentionally not updated — preserved from DB

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

const maxRecipeImageBytes = 2 * 1024 * 1024 // 2 MB

func (handler *RecipeHandler) ServeImage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	recipeID := chi.URLParam(r, "id")

	imageData, err := handler.recipeRepo.FindImageData(ctx, recipeID)
	if err != nil || imageData == "" {
		http.NotFound(w, r)
		return
	}

	withoutPrefix, ok := strings.CutPrefix(imageData, "data:")
	if !ok {
		http.NotFound(w, r)
		return
	}
	parts := strings.SplitN(withoutPrefix, ";base64,", 2)
	if len(parts) != 2 {
		http.NotFound(w, r)
		return
	}

	imageBytes, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		slog.Error("decoding recipe image", "error", err)
		http.Error(w, "Corrupted image data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", parts[0])
	w.WriteHeader(http.StatusOK)
	w.Write(imageBytes)
}

func (handler *RecipeHandler) UploadImage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	recipeID := chi.URLParam(r, "id")

	if _, err := handler.recipeRepo.FindByID(ctx, recipeID); err != nil {
		http.NotFound(w, r)
		return
	}

	if err := r.ParseMultipartForm(maxRecipeImageBytes + 1024); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "Missing image file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	imageBytes, err := io.ReadAll(io.LimitReader(file, maxRecipeImageBytes+1))
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusInternalServerError)
		return
	}
	if len(imageBytes) > maxRecipeImageBytes {
		http.Error(w, "Image exceeds 2 MB limit", http.StatusBadRequest)
		return
	}

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(imageBytes)
	}

	dataURI := "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(imageBytes)

	if err := handler.recipeRepo.UpdateImage(ctx, recipeID, dataURI); err != nil {
		slog.Error("updating recipe image", "error", err)
		http.Error(w, "Failed to save image", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/recipes/%s", recipeID), http.StatusFound)
}

func (handler *RecipeHandler) RemoveImage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	recipeID := chi.URLParam(r, "id")

	if err := handler.recipeRepo.ClearImage(ctx, recipeID); err != nil {
		slog.Error("clearing recipe image", "error", err)
		http.Error(w, "Failed to remove image", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/recipes/%s", recipeID), http.StatusFound)
}

func (handler *RecipeHandler) Step(w http.ResponseWriter, r *http.Request) {
	indexStr := r.URL.Query().Get("index")
	index, err := strconv.Atoi(indexStr)
	if err != nil {
		index = 0
	}
	component := pages.RecipeStepField(index, "")
	component.Render(r.Context(), w)
}

func parseMealType(value string) *models.RecipeMealType {
	switch models.RecipeMealType(value) {
	case models.RecipeMealTypeBreakfast, models.RecipeMealTypeLunch,
		models.RecipeMealTypeDinner, models.RecipeMealTypeSide, models.RecipeMealTypeDessert:
		mt := models.RecipeMealType(value)
		return &mt
	}
	return nil
}

func parseSteps(r *http.Request) []string {
	var steps []string
	for i := 0; ; i++ {
		step := strings.TrimSpace(r.FormValue(fmt.Sprintf("step_%d", i)))
		if step == "" {
			break
		}
		steps = append(steps, step)
	}
	return steps
}

func splitIntoSteps(text string) []string {
	var steps []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			steps = append(steps, line)
		}
	}
	return steps
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
