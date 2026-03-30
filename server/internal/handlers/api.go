package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/bensuskins/family-hub/internal/middleware"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/services"
	"github.com/go-chi/chi/v5"
)

type APIHandler struct {
	choreRepo       repository.ChoreRepository
	userRepo        repository.UserRepository
	categoryRepo    repository.CategoryRepository
	assignmentRepo  repository.ChoreAssignmentRepository
	tokenRepo       repository.APITokenRepository
	choreService    *services.ChoreService
	mealPlanRepo    repository.MealPlanRepository
	recipeRepo      repository.RecipeRepository
	icalFetcher     *services.ICalFetcher
	oidcUserInfoURL string
	clientID        string
	oidcIssuer      string
}

func NewAPIHandler(
	choreRepo repository.ChoreRepository,
	userRepo repository.UserRepository,
	categoryRepo repository.CategoryRepository,
	assignmentRepo repository.ChoreAssignmentRepository,
	tokenRepo repository.APITokenRepository,
	choreService *services.ChoreService,
	mealPlanRepo repository.MealPlanRepository,
	recipeRepo repository.RecipeRepository,
	icalFetcher *services.ICalFetcher,
	oidcUserInfoURL string,
	clientID string,
	oidcIssuer string,
) *APIHandler {
	return &APIHandler{
		choreRepo:       choreRepo,
		userRepo:        userRepo,
		categoryRepo:    categoryRepo,
		assignmentRepo:  assignmentRepo,
		tokenRepo:       tokenRepo,
		choreService:    choreService,
		mealPlanRepo:    mealPlanRepo,
		recipeRepo:      recipeRepo,
		icalFetcher:     icalFetcher,
		oidcUserInfoURL: oidcUserInfoURL,
		clientID:        clientID,
		oidcIssuer:      oidcIssuer,
	}
}

func (handler *APIHandler) ClientConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"clientID": handler.clientID,
		"issuer":   handler.oidcIssuer,
	})
}

func (handler *APIHandler) ExchangeToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing bearer token"})
		return
	}
	oidcToken := strings.TrimPrefix(authHeader, "Bearer ")

	req, err := http.NewRequestWithContext(ctx, "GET", handler.oidcUserInfoURL, nil)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	req.Header.Set("Authorization", "Bearer "+oidcToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid token"})
		return
	}
	defer resp.Body.Close()

	var userInfo struct {
		Sub               string `json:"sub"`
		Email             string `json:"email"`
		Name              string `json:"name"`
		PreferredUsername string `json:"preferred_username"`
		Picture           string `json:"picture"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil || userInfo.Sub == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid userinfo response"})
		return
	}

	user, err := handler.userRepo.FindByOIDCSubject(ctx, userInfo.Sub)
	if errors.Is(err, sql.ErrNoRows) {
		name := userInfo.Name
		if name == "" {
			name = userInfo.PreferredUsername
		}
		if name == "" {
			name = userInfo.Email
		}

		count, _ := handler.userRepo.Count(ctx)
		role := models.RoleMember
		if count == 0 {
			role = models.RoleAdmin
		}

		user, err = handler.userRepo.Create(ctx, models.User{
			OIDCSubject: userInfo.Sub,
			Email:       userInfo.Email,
			Name:        name,
			AvatarURL:   userInfo.Picture,
			Role:        role,
		})
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create user"})
			return
		}
	} else if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "user lookup failed"})
		return
	}

	// Replace any existing iOS tokens for this user
	existing, _ := handler.tokenRepo.FindByUserIDAndName(ctx, user.ID, "iOS App")
	for _, t := range existing {
		_ = handler.tokenRepo.Delete(ctx, t.ID)
	}

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to generate token"})
		return
	}
	plainToken := hex.EncodeToString(tokenBytes)

	if _, err := handler.tokenRepo.Create(ctx, models.APIToken{
		Name:            "iOS App",
		TokenHash:       repository.HashToken(plainToken),
		Scope:           models.TokenScopeAPI,
		CreatedByUserID: user.ID,
	}); err != nil {
		slog.Error("creating iOS API token", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create token"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"token": plainToken,
		"user":  user,
	})
}

func (handler *APIHandler) Me(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUser(ctx)
	writeJSON(w, http.StatusOK, user)
}

func (handler *APIHandler) ListChores(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	filter := repository.ChoreFilter{}

	if status := r.URL.Query().Get("status"); status != "" {
		s := models.ChoreStatus(status)
		filter.Status = &s
	}
	if assignedTo := r.URL.Query().Get("assigned_to"); assignedTo != "" {
		filter.AssignedToUser = &assignedTo
	}

	chores, err := handler.choreRepo.FindAll(ctx, filter)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load chores"})
		return
	}
	writeJSON(w, http.StatusOK, chores)
}

func (handler *APIHandler) GetChore(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	chore, err := handler.choreRepo.FindByID(ctx, chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "chore not found"})
		return
	}
	writeJSON(w, http.StatusOK, chore)
}

func (handler *APIHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	users, err := handler.userRepo.FindAll(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load users"})
		return
	}
	writeJSON(w, http.StatusOK, users)
}

func (handler *APIHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, err := handler.userRepo.FindByID(ctx, chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (handler *APIHandler) ListCategories(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	categories, err := handler.categoryRepo.FindAll(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load categories"})
		return
	}
	writeJSON(w, http.StatusOK, categories)
}

func (handler *APIHandler) DashboardStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	now := time.Now()

	choresDueToday, err := handler.choreRepo.FindDueToday(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load chores due today"})
		return
	}
	overdueChores, err := handler.choreRepo.FindOverdueChores(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load overdue chores"})
		return
	}

	if choresDueToday == nil {
		choresDueToday = []models.Chore{}
	}
	if overdueChores == nil {
		overdueChores = []models.Chore{}
	}

	mealsThisWeek := []models.MealPlan{}
	todayMeals := []models.MealPlan{}
	if handler.mealPlanRepo != nil {
		weekStart := now.Truncate(24 * time.Hour)
		weekEnd := weekStart.AddDate(0, 0, 7)
		if meals, err := handler.mealPlanRepo.FindAll(ctx, repository.MealPlanFilter{
			DateFrom: weekStart.Format("2006-01-02"),
			DateTo:   weekEnd.Format("2006-01-02"),
		}); err == nil && meals != nil {
			mealsThisWeek = meals
		}
		if meals, err := handler.mealPlanRepo.FindByDate(ctx, now.Format("2006-01-02")); err == nil && meals != nil {
			todayMeals = meals
		}
	}

	stats := map[string]interface{}{
		"chores_due_today":      len(choresDueToday),
		"chores_overdue":        len(overdueChores),
		"chores_due_today_list": choresDueToday,
		"chores_overdue_list":   overdueChores,
		"meals_this_week":       len(mealsThisWeek),
		"today_meals":           todayMeals,
	}
	writeJSON(w, http.StatusOK, stats)
}

func (handler *APIHandler) CreateToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUser(ctx)

	name := r.FormValue("name")
	if name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}

	rawToken := generateToken()
	token := models.APIToken{
		Name:            name,
		TokenHash:       repository.HashToken(rawToken),
		CreatedByUserID: user.ID,
	}

	created, err := handler.tokenRepo.Create(ctx, token)
	if err != nil {
		slog.Error("creating token", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create token"})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"id":    created.ID,
		"name":  created.Name,
		"token": rawToken,
	})
}

func (handler *APIHandler) DeleteToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	if err := handler.tokenRepo.Delete(ctx, id); err != nil {
		slog.Error("deleting token", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete token"})
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (handler *APIHandler) CompleteChore(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUser(ctx)
	choreID := chi.URLParam(r, "id")

	if err := handler.choreService.CompleteChore(ctx, choreID, user.ID); err != nil {
		switch {
		case errors.Is(err, services.ErrChoreAlreadyComplete):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "chore is already complete"})
		case errors.Is(err, sql.ErrNoRows):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "chore not found"})
		default:
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to complete chore"})
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (handler *APIHandler) ListMeals(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	weekParam := r.URL.Query().Get("week")
	if weekParam == "" {
		weekParam = time.Now().Format("2006-01-02")
	}

	weekStart, err := time.Parse("2006-01-02", weekParam)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid week format, use YYYY-MM-DD"})
		return
	}

	// Snap to Monday (weekday 0=Sunday, 1=Monday, ..., 6=Saturday)
	offset := (int(weekStart.Weekday()) - int(time.Monday) + 7) % 7
	weekStart = weekStart.AddDate(0, 0, -offset)
	weekEnd := weekStart.AddDate(0, 0, 6)

	meals, err := handler.mealPlanRepo.FindAll(ctx, repository.MealPlanFilter{
		DateFrom: weekStart.Format("2006-01-02"),
		DateTo:   weekEnd.Format("2006-01-02"),
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load meals"})
		return
	}
	if meals == nil {
		meals = []models.MealPlan{}
	}

	writeJSON(w, http.StatusOK, meals)
}

func (handler *APIHandler) ListCalendar(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	now := time.Now()

	view := r.URL.Query().Get("view")
	if view == "" {
		view = "month"
	}

	var start, end time.Time
	switch view {
	case "week":
		date := now
		if dateStr := r.URL.Query().Get("date"); dateStr != "" {
			if d, err := time.Parse("2006-01-02", dateStr); err == nil {
				date = d
			}
		}
		offset := (int(date.Weekday()) + 6) % 7
		start = time.Date(date.Year(), date.Month(), date.Day()-offset, 0, 0, 0, 0, time.Local)
		end = start.AddDate(0, 0, 7)

	case "day":
		date := now
		if dateStr := r.URL.Query().Get("date"); dateStr != "" {
			if d, err := time.Parse("2006-01-02", dateStr); err == nil {
				date = d
			}
		}
		start = time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.Local)
		end = start.AddDate(0, 0, 1)

	default: // "month"
		monthParam := r.URL.Query().Get("month")
		if monthParam == "" {
			monthParam = now.Format("2006-01")
		}
		monthStart, err := time.Parse("2006-01", monthParam)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid month format, use YYYY-MM"})
			return
		}
		start = monthStart
		end = monthStart.AddDate(0, 1, -1)
	}

	chores, err := handler.choreRepo.FindAll(ctx, repository.ChoreFilter{
		DueAfter:  &start,
		DueBefore: &end,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load chores"})
		return
	}
	if chores == nil {
		chores = []models.Chore{}
	}

	var events []models.Event
	if handler.icalFetcher != nil {
		fetchedEvents, err := handler.icalFetcher.FetchForRange(ctx, start, end)
		if err != nil {
			slog.Error("fetching ical events for calendar API", "error", err)
		} else {
			events = fetchedEvents
		}
	}
	if events == nil {
		events = []models.Event{}
	}

	var meals []models.MealPlan
	if handler.mealPlanRepo != nil {
		fetchedMeals, err := handler.mealPlanRepo.FindAll(ctx, repository.MealPlanFilter{
			DateFrom: start.Format("2006-01-02"),
			DateTo:   end.Format("2006-01-02"),
		})
		if err != nil {
			slog.Error("fetching meals for calendar API", "error", err)
		} else {
			meals = fetchedMeals
		}
	}
	if meals == nil {
		meals = []models.MealPlan{}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"chores": chores,
		"events": events,
		"meals":  meals,
	})
}

func (handler *APIHandler) ListRecipes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	recipes, err := handler.recipeRepo.FindAll(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load recipes"})
		return
	}
	if recipes == nil {
		recipes = []models.Recipe{}
	}
	writeJSON(w, http.StatusOK, recipes)
}

func (handler *APIHandler) GetRecipe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	recipe, err := handler.recipeRepo.FindByID(ctx, chi.URLParam(r, "id"))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "recipe not found"})
		} else {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load recipe"})
		}
		return
	}
	writeJSON(w, http.StatusOK, recipe)
}

func (handler *APIHandler) SaveMeal(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUser(ctx)

	var body struct {
		Date     string `json:"date"`
		MealType string `json:"mealType"`
		Name     string `json:"name"`
		RecipeID string `json:"recipeID,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}

	if body.Date == "" || body.MealType == "" || body.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "date, mealType, and name are required"})
		return
	}

	meal := models.MealPlan{
		Date:            body.Date,
		MealType:        models.MealType(body.MealType),
		Name:            body.Name,
		CreatedByUserID: user.ID,
	}
	if body.RecipeID != "" {
		meal.RecipeID = &body.RecipeID
	}

	if err := handler.mealPlanRepo.Upsert(ctx, meal); err != nil {
		slog.Error("saving meal via API", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save meal"})
		return
	}

	saved, err := handler.mealPlanRepo.FindByDateAndType(ctx, body.Date, models.MealType(body.MealType))
	if err != nil {
		slog.Error("finding saved meal via API", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to retrieve saved meal"})
		return
	}

	writeJSON(w, http.StatusOK, saved)
}

func (handler *APIHandler) DeleteMeal(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	date := r.URL.Query().Get("date")
	mealType := r.URL.Query().Get("mealType")

	if date == "" || mealType == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "date and mealType query params are required"})
		return
	}

	if err := handler.mealPlanRepo.Delete(ctx, date, models.MealType(mealType)); err != nil {
		slog.Error("deleting meal via API", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete meal"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (handler *APIHandler) CreateRecipe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUser(ctx)

	var body struct {
		Title       string                  `json:"title"`
		Steps       []string                `json:"steps"`
		Ingredients []models.IngredientGroup `json:"ingredients"`
		MealType    string                  `json:"mealType,omitempty"`
		Servings    *int                    `json:"servings,omitempty"`
		PrepTime    *string                 `json:"prepTime,omitempty"`
		CookTime    *string                 `json:"cookTime,omitempty"`
		SourceURL   *string                 `json:"sourceURL,omitempty"`
		ImageData   *string                 `json:"imageData,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}

	if body.Title == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "title is required"})
		return
	}

	recipe := models.Recipe{
		Title:           body.Title,
		Steps:           body.Steps,
		Ingredients:     body.Ingredients,
		Servings:        body.Servings,
		PrepTime:        body.PrepTime,
		CookTime:        body.CookTime,
		SourceURL:       body.SourceURL,
		CreatedByUserID: user.ID,
	}
	if body.MealType != "" {
		mt := models.RecipeMealType(body.MealType)
		recipe.MealType = &mt
	}

	created, err := handler.recipeRepo.Create(ctx, recipe)
	if err != nil {
		slog.Error("creating recipe via API", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create recipe"})
		return
	}

	if body.ImageData != nil && *body.ImageData != "" {
		if err := handler.recipeRepo.UpdateImage(ctx, created.ID, *body.ImageData); err != nil {
			slog.Error("saving recipe image via API", "error", err)
		} else {
			created.HasImage = true
		}
	}

	writeJSON(w, http.StatusCreated, created)
}

func (handler *APIHandler) UpdateRecipe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	recipeID := chi.URLParam(r, "id")

	existing, err := handler.recipeRepo.FindByID(ctx, recipeID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "recipe not found"})
		} else {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load recipe"})
		}
		return
	}

	var body struct {
		Title       string                   `json:"title"`
		Steps       []string                 `json:"steps"`
		Ingredients []models.IngredientGroup `json:"ingredients"`
		MealType    string                   `json:"mealType,omitempty"`
		Servings    *int                     `json:"servings,omitempty"`
		PrepTime    *string                  `json:"prepTime,omitempty"`
		CookTime    *string                  `json:"cookTime,omitempty"`
		SourceURL   *string                  `json:"sourceURL,omitempty"`
		ImageData   *string                  `json:"imageData,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}

	if body.Title == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "title is required"})
		return
	}

	existing.Title = body.Title
	existing.Steps = body.Steps
	existing.Ingredients = body.Ingredients
	existing.Servings = body.Servings
	existing.PrepTime = body.PrepTime
	existing.CookTime = body.CookTime
	existing.SourceURL = body.SourceURL
	if body.MealType != "" {
		mt := models.RecipeMealType(body.MealType)
		existing.MealType = &mt
	} else {
		existing.MealType = nil
	}

	if err := handler.recipeRepo.Update(ctx, existing); err != nil {
		slog.Error("updating recipe via API", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update recipe"})
		return
	}

	if body.ImageData != nil {
		if *body.ImageData == "" {
			handler.recipeRepo.ClearImage(ctx, recipeID)
		} else {
			handler.recipeRepo.UpdateImage(ctx, recipeID, *body.ImageData)
		}
	}

	updated, err := handler.recipeRepo.FindByID(ctx, recipeID)
	if err != nil {
		writeJSON(w, http.StatusOK, existing)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (handler *APIHandler) DeleteRecipe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	recipeID := chi.URLParam(r, "id")

	if _, err := handler.recipeRepo.FindByID(ctx, recipeID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "recipe not found"})
		} else {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load recipe"})
		}
		return
	}

	if err := handler.mealPlanRepo.ClearRecipeID(ctx, recipeID); err != nil {
		slog.Error("clearing recipe from meal plans via API", "error", err)
	}

	if err := handler.recipeRepo.Delete(ctx, recipeID); err != nil {
		slog.Error("deleting recipe via API", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete recipe"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func generateToken() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
