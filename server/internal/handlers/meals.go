package handlers

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/bensuskins/family-hub/internal/middleware"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/templates/pages"
)

type MealHandler struct {
	mealPlanRepo repository.MealPlanRepository
	recipeRepo   repository.RecipeRepository
}

func NewMealHandler(mealPlanRepo repository.MealPlanRepository, recipeRepo repository.RecipeRepository) *MealHandler {
	return &MealHandler{mealPlanRepo: mealPlanRepo, recipeRepo: recipeRepo}
}

func (handler *MealHandler) Planner(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUser(ctx)

	weekStart := lastFriday(time.Now())
	if weekStartStr := r.URL.Query().Get("week_start"); weekStartStr != "" {
		if parsed, err := time.Parse(DateFormat, weekStartStr); err == nil {
			weekStart = parsed
		}
	}

	weekEnd := weekStart.AddDate(0, 0, 6)

	meals, err := handler.mealPlanRepo.FindAll(ctx, repository.MealPlanFilter{
		DateFrom: weekStart.Format(DateFormat),
		DateTo:   weekEnd.Format(DateFormat),
	})
	if err != nil {
		slog.Error("finding meals for planner", "error", err)
	}

	recipes, err := handler.recipeRepo.FindAll(ctx)
	if err != nil {
		slog.Error("finding recipes for planner", "error", err)
	}

	mealMap := make(map[string]models.MealPlan)
	for _, meal := range meals {
		mealMap[meal.Date+"-"+string(meal.MealType)] = meal
	}

	var days []time.Time
	for i := 0; i < 7; i++ {
		days = append(days, weekStart.AddDate(0, 0, i))
	}

	pages.MealPlanner(pages.MealPlannerProps{
		User:      user,
		WeekStart: weekStart,
		Days:      days,
		MealMap:   mealMap,
		Recipes:   recipes,
	}).Render(ctx, w)
}

func (handler *MealHandler) SaveMeal(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUser(ctx)

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	date := r.FormValue("date")
	mealType := models.MealType(r.FormValue("meal_type"))
	name := r.FormValue("name")
	recipeID := r.FormValue("recipe_id")

	if name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	meal := models.MealPlan{
		Date:            date,
		MealType:        mealType,
		Name:            name,
		CreatedByUserID: user.ID,
	}
	if recipeID != "" {
		meal.RecipeID = &recipeID
	}

	if err := handler.mealPlanRepo.Upsert(ctx, meal); err != nil {
		slog.Error("saving meal", "error", err)
		http.Error(w, "Error saving meal", http.StatusInternalServerError)
		return
	}

	saved, err := handler.mealPlanRepo.FindByDateAndType(ctx, date, mealType)
	if err != nil {
		slog.Error("finding saved meal", "error", err)
	}

	pages.MealSlotOOB(date, mealType, &saved).Render(ctx, w)
}

func (handler *MealHandler) DeleteMeal(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	date := r.FormValue("date")
	mealType := models.MealType(r.FormValue("meal_type"))

	if err := handler.mealPlanRepo.Delete(ctx, date, mealType); err != nil {
		slog.Error("deleting meal", "error", err)
		http.Error(w, "Error deleting meal", http.StatusInternalServerError)
		return
	}

	pages.MealSlotOOB(date, mealType, nil).Render(ctx, w)
}

func (handler *MealHandler) Dismiss(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (handler *MealHandler) Cell(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	date := r.URL.Query().Get("date")
	mealType := models.MealType(r.URL.Query().Get("meal_type"))
	editMode := r.URL.Query().Get("edit") == "true"

	var meal *models.MealPlan
	if found, err := handler.mealPlanRepo.FindByDateAndType(ctx, date, mealType); err == nil {
		meal = &found
	}

	if editMode {
		pages.MealEditDrawer(date, mealType, meal).Render(ctx, w)
	} else {
		pages.MealSlotContent(date, mealType, meal).Render(ctx, w)
	}
}

func (handler *MealHandler) RecipePicker(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	date := r.URL.Query().Get("date")
	mealType := models.MealType(r.URL.Query().Get("meal_type"))
	query := r.URL.Query().Get("q")
	selectID := r.URL.Query().Get("select")

	recipes, err := handler.recipeRepo.FindAll(ctx)
	if err != nil {
		slog.Error("finding recipes for picker", "error", err)
	}

	// If a recipe was selected, return to drawer state with that recipe pre-filled
	if selectID != "" {
		for _, recipe := range recipes {
			if recipe.ID == selectID {
				pages.MealEditDrawer(date, mealType, &models.MealPlan{
					Date:     date,
					MealType: mealType,
					Name:     recipe.Title,
					RecipeID: &recipe.ID,
				}).Render(ctx, w)
				return
			}
		}
	}

	// Filter by query if provided
	if query != "" {
		lower := strings.ToLower(query)
		var filtered []models.Recipe
		for _, recipe := range recipes {
			if strings.Contains(strings.ToLower(recipe.Title), lower) {
				filtered = append(filtered, recipe)
			}
		}
		recipes = filtered
	}

	pages.MealRecipePicker(date, mealType, query, recipes).Render(ctx, w)
}

// lastFriday returns the most recent Friday at midnight local time (or today if already Friday).
func lastFriday(t time.Time) time.Time {
	offset := (int(t.Weekday()) - int(time.Friday) + 7) % 7
	return time.Date(t.Year(), t.Month(), t.Day()-offset, 0, 0, 0, 0, t.Location())
}
