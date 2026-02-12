package handlers

import (
	"log/slog"
	"net/http"
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

	now := time.Now()
	weekStart := now
	if weekStartStr := r.URL.Query().Get("week_start"); weekStartStr != "" {
		if parsed, err := time.Parse("2006-01-02", weekStartStr); err == nil {
			weekStart = parsed
		}
	} else {
		weekday := int(weekStart.Weekday())
		weekStart = time.Date(weekStart.Year(), weekStart.Month(), weekStart.Day()-weekday, 0, 0, 0, 0, time.Local)
	}

	weekEnd := weekStart.AddDate(0, 0, 6)

	meals, err := handler.mealPlanRepo.FindAll(ctx, repository.MealPlanFilter{
		DateFrom: weekStart.Format("2006-01-02"),
		DateTo:   weekEnd.Format("2006-01-02"),
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

	component := pages.MealPlanner(pages.MealPlannerProps{
		User:      user,
		WeekStart: weekStart,
		Days:      days,
		MealMap:   mealMap,
		Recipes:   recipes,
	})
	component.Render(ctx, w)
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
	notes := r.FormValue("notes")
	recipeID := r.FormValue("recipe_id")

	if name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	meal := models.MealPlan{
		Date:            date,
		MealType:        mealType,
		Name:            name,
		Notes:           notes,
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

	recipes, err := handler.recipeRepo.FindAll(ctx)
	if err != nil {
		slog.Error("finding recipes", "error", err)
	}

	saved, err := handler.mealPlanRepo.FindByDateAndType(ctx, date, mealType)
	if err != nil {
		slog.Error("finding saved meal", "error", err)
	}

	w.Header().Set("HX-Trigger", "closeMealModal")
	component := pages.MealCell(date, mealType, &saved, recipes)
	component.Render(ctx, w)
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

	recipes, err := handler.recipeRepo.FindAll(ctx)
	if err != nil {
		slog.Error("finding recipes", "error", err)
	}

	w.Header().Set("HX-Trigger", "closeMealModal")
	component := pages.MealCell(date, mealType, nil, recipes)
	component.Render(ctx, w)
}

func (handler *MealHandler) Cell(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	date := r.URL.Query().Get("date")
	mealType := models.MealType(r.URL.Query().Get("meal_type"))
	editMode := r.URL.Query().Get("edit") == "true"

	recipes, err := handler.recipeRepo.FindAll(ctx)
	if err != nil {
		slog.Error("finding recipes", "error", err)
	}

	var meal *models.MealPlan
	if found, err := handler.mealPlanRepo.FindByDateAndType(ctx, date, mealType); err == nil {
		meal = &found
	}

	if editMode {
		component := pages.MealCellEdit(date, mealType, meal, recipes)
		component.Render(ctx, w)
	} else {
		component := pages.MealCell(date, mealType, meal, recipes)
		component.Render(ctx, w)
	}
}
