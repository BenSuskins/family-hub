package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	categoryapp "github.com/bensuskins/family-hub/internal/application/categories"
	dashboardapp "github.com/bensuskins/family-hub/internal/application/dashboard"
	householdapp "github.com/bensuskins/family-hub/internal/application/household"
	identityapp "github.com/bensuskins/family-hub/internal/application/identity"
	userapp "github.com/bensuskins/family-hub/internal/application/users"

	recipeapp "github.com/bensuskins/family-hub/internal/application/recipes"

	calendarapp "github.com/bensuskins/family-hub/internal/application/calendar"

	choreapp "github.com/bensuskins/family-hub/internal/application/chores"
	inventoryapp "github.com/bensuskins/family-hub/internal/application/inventory"
	mealapp "github.com/bensuskins/family-hub/internal/application/meals"
	"github.com/bensuskins/family-hub/internal/middleware"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/observability"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/go-chi/chi/v5"
)

type APIHandler struct {
	users            *userapp.Service
	categories       *categoryapp.Service
	calendarView     *calendarapp.ViewService
	tokens           *identityapp.TokenAdministration
	settingsRepo     repository.SettingsRepository
	generalSettings  *householdapp.Service
	dashboard        *dashboardapp.Service
	choreService     *choreapp.ChoreService
	mealService      *mealapp.Service
	recipes          *recipeapp.Catalogue
	inventoryService *inventoryapp.Service
	stockService     *inventoryapp.StockService
	icalFetcher      *calendarapp.Service
	recipeExtractor  *recipeapp.Service
	identity         *identityapp.Service
	clientID         string
	oidcIssuer       string
}

const (
	maxJSONRequestBytes int64 = 10 << 20
)

func NewAPIHandler(
	users *userapp.Service,
	categories *categoryapp.Service,
	calendarView *calendarapp.ViewService,
	tokens *identityapp.TokenAdministration,
	settingsRepo repository.SettingsRepository,
	generalSettings *householdapp.Service,
	choreService *choreapp.ChoreService,
	mealPlanRepo repository.MealPlanRepository,
	recipes *recipeapp.Catalogue,
	inventoryRepo interface {
		inventoryapp.Store
		inventoryapp.StockStore
	},
	icalFetcher *calendarapp.Service,
	recipeExtractor *recipeapp.Service,
	identity *identityapp.Service,
	dashboard *dashboardapp.Service,
	clientID string,
	oidcIssuer string,
) *APIHandler {
	var mealService *mealapp.Service
	if mealPlanRepo != nil {
		mealService = mealapp.New(mealPlanRepo)
	}
	return &APIHandler{
		users:            users,
		categories:       categories,
		calendarView:     calendarView,
		tokens:           tokens,
		settingsRepo:     settingsRepo,
		generalSettings:  generalSettings,
		choreService:     choreService,
		mealService:      mealService,
		recipes:          recipes,
		inventoryService: inventoryapp.New(inventoryRepo),
		stockService:     inventoryapp.NewStockService(inventoryRepo),
		icalFetcher:      icalFetcher,
		recipeExtractor:  recipeExtractor,
		identity:         identity,
		dashboard:        dashboard,
		clientID:         clientID,
		oidcIssuer:       oidcIssuer,
	}
}

func (handler *APIHandler) ClientConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"clientID": handler.clientID,
		"issuer":   handler.oidcIssuer,
	})
}

func (handler *APIHandler) ExchangeToken(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		writeJSONError(w, http.StatusUnauthorized, "missing bearer token")
		return
	}
	result, err := handler.identity.Exchange(r.Context(), strings.TrimPrefix(authHeader, "Bearer "), r.URL.Query().Get("client"))
	if err != nil {
		switch {
		case errors.Is(err, identityapp.ErrInvalidToken):
			writeJSONError(w, http.StatusUnauthorized, "invalid token")
		case errors.Is(err, identityapp.ErrProviderRequest):
			writeAPIError(w, r, http.StatusInternalServerError, "internal error", err)
		case errors.Is(err, identityapp.ErrProviderUnavailable):
			writeAPIError(w, r, http.StatusBadGateway, "identity provider unavailable", err)
		case errors.Is(err, identityapp.ErrProviderResponse):
			writeAPIError(w, r, http.StatusBadGateway, "invalid identity provider response", err)
		case errors.Is(err, identityapp.ErrLookup):
			writeAPIError(w, r, http.StatusInternalServerError, "failed to look up user", err)
		case errors.Is(err, identityapp.ErrCount):
			writeAPIError(w, r, http.StatusInternalServerError, "failed to determine user role", err)
		case errors.Is(err, identityapp.ErrCreateUser):
			writeAPIError(w, r, http.StatusInternalServerError, "failed to create user", err)
		case errors.Is(err, identityapp.ErrReplaceToken):
			writeAPIError(w, r, http.StatusInternalServerError, "failed to replace client token", err)
		case errors.Is(err, identityapp.ErrGenerateToken):
			writeAPIError(w, r, http.StatusInternalServerError, "failed to generate token", err)
		default:
			writeAPIError(w, r, http.StatusInternalServerError, "failed to create token", err)
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"token": result.Token, "user": result.User})
}

func (handler *APIHandler) Me(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUser(ctx)
	writeJSON(w, http.StatusOK, user)
}

func (handler *APIHandler) ListChores(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	filter := choreapp.ChoreFilter{}

	if status := r.URL.Query().Get("status"); status != "" {
		s := models.ChoreStatus(status)
		filter.Status = &s
	}
	if assignedTo := r.URL.Query().Get("assigned_to"); assignedTo != "" {
		filter.AssignedToUser = &assignedTo
	}

	chores, err := handler.choreService.FindChores(ctx, filter)
	if err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "failed to load chores", err)
		return
	}
	if chores == nil {
		chores = []models.Chore{}
	}
	writeJSON(w, http.StatusOK, chores)
}

func (handler *APIHandler) GetChore(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	chore, err := handler.choreService.FindChore(ctx, chi.URLParam(r, "id"))
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "chore not found")
		return
	}
	if chore.Status == models.ChoreStatusSkipped || chore.Status == models.ChoreStatusSuperseded {
		writeJSONError(w, 409, "chore is no longer outstanding")
		return
	}
	writeJSON(w, http.StatusOK, chore)
}

func (handler *APIHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	users, err := handler.users.List(ctx)
	if err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "failed to load users", err)
		return
	}
	writeJSON(w, http.StatusOK, mapUsers(users))
}

func (handler *APIHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, err := handler.users.Find(ctx, chi.URLParam(r, "id"))
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "user not found")
		return
	}
	writeJSON(w, http.StatusOK, mapUser(user))
}

func (handler *APIHandler) ListCategories(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	categories, err := handler.categories.List(ctx)
	if err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "failed to load categories", err)
		return
	}
	writeJSON(w, http.StatusOK, mapCategories(categories))
}

func (handler *APIHandler) DashboardStats(w http.ResponseWriter, r *http.Request) {
	stats, err := handler.dashboard.Stats(r.Context())
	if err != nil {
		message := "failed to load due chores"
		switch {
		case errors.Is(err, dashboardapp.ErrLocation):
			message = "failed to load household time zone"
		case errors.Is(err, dashboardapp.ErrBins):
			message = "failed to load bin schedule"
		case errors.Is(err, dashboardapp.ErrWeeklyMeals):
			message = "failed to load weekly meals"
		case errors.Is(err, dashboardapp.ErrTodayMeals):
			message = "failed to load today's meals"
		}
		writeAPIError(w, r, http.StatusInternalServerError, message, err)
		return
	}
	writeJSON(w, http.StatusOK, mapDashboard(stats))
}

func (handler *APIHandler) CreateToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUser(ctx)

	var name string
	if ct := r.Header.Get("Content-Type"); strings.Contains(ct, "application/json") {
		var body struct {
			Name string `json:"name"`
		}
		if !decodeJSONBody(w, r, &body) {
			return
		}
		name = body.Name
	} else {
		name = r.FormValue("name")
	}
	if name == "" {
		writeJSONError(w, http.StatusBadRequest, "name is required")
		return
	}

	created, err := handler.tokens.Create(ctx, name, user.ID)
	if err != nil {
		message := "failed to create token"
		if errors.Is(err, identityapp.ErrGenerateToken) {
			message = "failed to generate token"
		}
		writeAPIError(w, r, http.StatusInternalServerError, message, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]interface{}{"id": created.ID, "name": created.Name, "token": created.Token})
}

func (handler *APIHandler) DeleteToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	if err := handler.tokens.Delete(ctx, id); err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "failed to delete token", err)
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
		case errors.Is(err, choreapp.ErrChoreAlreadyComplete):
			writeJSONError(w, http.StatusConflict, "chore is already complete")
		case errors.Is(err, choreapp.ErrChoreNotFound):
			writeJSONError(w, http.StatusNotFound, "chore not found")
		default:
			writeAPIError(w, r, http.StatusInternalServerError, "failed to complete chore", err)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (handler *APIHandler) ListMeals(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	weekParam := r.URL.Query().Get("week")
	if weekParam == "" {
		weekParam = time.Now().Format(DateFormat)
	}

	weekStart, err := time.Parse(DateFormat, weekParam)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid week format, use YYYY-MM-DD")
		return
	}

	weekEnd := weekStart.AddDate(0, 0, 6)

	meals, err := handler.mealService.FindAll(ctx, mealapp.Filter{
		DateFrom: weekStart.Format(DateFormat),
		DateTo:   weekEnd.Format(DateFormat),
	})
	if err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "failed to load meals", err)
		return
	}
	if meals == nil {
		meals = []models.MealPlan{}
	}

	writeJSON(w, http.StatusOK, meals)
}

func (handler *APIHandler) ListCalendar(w http.ResponseWriter, r *http.Request) {
	query := calendarapp.ViewQuery{View: r.URL.Query().Get("view"), Date: r.URL.Query().Get("date"), Month: r.URL.Query().Get("month")}
	query.OnFailure = func(source calendarapp.ViewFailureSource, failure error) {
		switch source {
		case calendarapp.ViewFailureLocation:
			message := "loading household timezone for calendar API"
			if errors.Is(failure, calendarapp.ErrViewLocationDecode) {
				message = "decoding household timezone for calendar API"
			}
			slog.Error(message, "method", r.Method, "route", observability.RoutePattern(r), "status", http.StatusOK, "error", failure)
		case calendarapp.ViewFailureEvents:
			observability.RecordDependencyFailure(r, "calendar_feed")
			slog.Error("fetching calendar events for API", "method", r.Method, "route", observability.RoutePattern(r), "status", http.StatusOK, "error", failure)
		case calendarapp.ViewFailureMeals:
			slog.Error("fetching meals for calendar API", "method", r.Method, "route", observability.RoutePattern(r), "status", http.StatusOK, "error", failure)
		}
	}
	result, err := handler.calendarView.List(r.Context(), query)
	if err != nil {
		switch {
		case errors.Is(err, calendarapp.ErrViewDate):
			writeJSONError(w, http.StatusBadRequest, "invalid date format, use YYYY-MM-DD")
		case errors.Is(err, calendarapp.ErrViewMonth):
			writeJSONError(w, http.StatusBadRequest, "invalid month format, use YYYY-MM")
		case errors.Is(err, calendarapp.ErrViewForecast):
			writeAPIError(w, r, http.StatusInternalServerError, "failed to load chore forecast", err)
		default:
			writeAPIError(w, r, http.StatusInternalServerError, "failed to load chores", err)
		}
		return
	}
	writeJSON(w, http.StatusOK, mapCalendarView(result))
}

func availabilityStatus(available bool) string {
	if !available {
		return "unavailable"
	}
	return "available"
}

func (handler *APIHandler) ListRecipes(w http.ResponseWriter, r *http.Request) {
	items, err := handler.recipes.List(r.Context())
	if err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "failed to load recipes", err)
		return
	}
	responses := make([]recipeResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, mapRecipeResponse(item))
	}
	writeJSON(w, http.StatusOK, responses)
}

func (handler *APIHandler) GetRecipe(w http.ResponseWriter, r *http.Request) {
	item, err := handler.recipes.Detail(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		if errors.Is(err, recipeapp.ErrNotFound) {
			writeJSONError(w, http.StatusNotFound, "recipe not found")
		} else {
			writeAPIError(w, r, http.StatusInternalServerError, "failed to load recipe", err)
		}
		return
	}
	writeJSON(w, http.StatusOK, mapRecipeResponse(item))
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
	if !decodeJSONBody(w, r, &body) {
		return
	}

	if body.Date == "" || body.MealType == "" || body.Name == "" {
		writeJSONError(w, http.StatusBadRequest, "date, mealType, and name are required")
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

	saved, err := handler.mealService.Save(ctx, meal)
	if err != nil {
		if errors.Is(err, mealapp.ErrSave) {
			writeAPIError(w, r, http.StatusInternalServerError, "failed to save meal", err)
		} else {
			writeAPIError(w, r, http.StatusInternalServerError, "failed to retrieve saved meal", err)
		}
		return
	}
	writeJSON(w, http.StatusOK, saved)
}

func (handler *APIHandler) DeleteMeal(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	date := r.URL.Query().Get("date")
	mealType := r.URL.Query().Get("mealType")

	if date == "" || mealType == "" {
		writeJSONError(w, http.StatusBadRequest, "date and mealType query params are required")
		return
	}

	if err := handler.mealService.Delete(ctx, date, models.MealType(mealType)); err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "failed to delete meal", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// choreAPIBody is the JSON request body shared by CreateChore and UpdateChore.
// It exposes the chore options needed by the iOS app.
type choreAPIBody struct {
	Name                 string   `json:"name"`
	Description          string   `json:"description"`
	CategoryID           *string  `json:"categoryId,omitempty"`
	Assignees            []string `json:"assignees,omitempty"`
	DueDate              *string  `json:"dueDate,omitempty"`
	DueTime              *string  `json:"dueTime,omitempty"`
	RecurrenceType       string   `json:"recurrenceType,omitempty"`
	RecurrenceInterval   int      `json:"recurrenceInterval,omitempty"`
	RecurrenceDays       []string `json:"recurrenceDays,omitempty"`
	RecurrenceDayOfMonth int      `json:"recurrenceDayOfMonth,omitempty"`
	RecurrenceUnit       string   `json:"recurrenceUnit,omitempty"`
	RecurrenceUntil      *string  `json:"recurrenceUntil,omitempty"`
	RecurrenceCount      *int     `json:"recurrenceCount,omitempty"`
	RecurOnComplete      bool     `json:"recurOnComplete,omitempty"`
}

// applyTo writes the body's schedule, category and recurrence fields onto a
// chore. Name/Description are set by the caller. Absent optional fields clear
// their target so an edit can remove a value.
func (b choreAPIBody) applyTo(chore *models.Chore) {
	recurrenceType := models.RecurrenceNone
	if b.RecurrenceType != "" {
		recurrenceType = models.RecurrenceType(b.RecurrenceType)
	}
	chore.RecurrenceType = recurrenceType
	chore.RecurrenceValue = buildRecurrenceValueFrom(
		recurrenceType, b.RecurrenceInterval, b.RecurrenceDays, b.RecurrenceDayOfMonth, b.RecurrenceUnit,
	)
	chore.RecurOnComplete = b.RecurOnComplete

	if b.CategoryID != nil && *b.CategoryID != "" {
		chore.CategoryID = b.CategoryID
	} else {
		chore.CategoryID = nil
	}

	if b.DueTime != nil && *b.DueTime != "" {
		chore.DueTime = b.DueTime
	} else {
		chore.DueTime = nil
	}

	if b.DueDate != nil && *b.DueDate != "" {
		if dueDate, err := time.Parse(DateFormat, *b.DueDate); err == nil {
			chore.DueDate = &dueDate
		}
	} else {
		chore.DueDate = nil
	}

	if b.RecurrenceUntil != nil && *b.RecurrenceUntil != "" {
		if until, err := time.Parse(DateFormat, *b.RecurrenceUntil); err == nil {
			chore.RecurrenceUntil = &until
		}
	} else {
		chore.RecurrenceUntil = nil
	}

	if b.RecurrenceCount != nil && *b.RecurrenceCount > 0 {
		chore.RecurrenceCount = b.RecurrenceCount
	} else {
		chore.RecurrenceCount = nil
	}
}

func (handler *APIHandler) CreateChore(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUser(ctx)

	var body choreAPIBody
	if !decodeJSONBody(w, r, &body) {
		return
	}
	if body.Name == "" {
		writeJSONError(w, http.StatusBadRequest, "name is required")
		return
	}

	chore := models.Chore{
		Name:            body.Name,
		Description:     body.Description,
		CreatedByUserID: user.ID,
	}
	body.applyTo(&chore)

	created, err := handler.choreService.CreateChore(ctx, chore, body.Assignees)
	if err != nil {
		writeAPIError(w, r, 500, "failed to create chore", err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (handler *APIHandler) UpdateChore(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	choreID := chi.URLParam(r, "id")

	chore, err := handler.choreService.FindChore(ctx, choreID)
	if err != nil {
		if errors.Is(err, choreapp.ErrChoreNotFound) {
			writeJSONError(w, http.StatusNotFound, "chore not found")
		} else {
			writeAPIError(w, r, http.StatusInternalServerError, "failed to load chore", err)
		}
		return
	}

	if chore.Status == models.ChoreStatusSkipped || chore.Status == models.ChoreStatusSuperseded {
		writeJSONError(w, http.StatusConflict, "chore is no longer outstanding")
		return
	}
	var body choreAPIBody
	if !decodeJSONBody(w, r, &body) {
		return
	}
	if body.Name == "" {
		writeJSONError(w, http.StatusBadRequest, "name is required")
		return
	}

	chore.Name = body.Name
	chore.Description = body.Description
	body.applyTo(&chore)
	updated, err := handler.choreService.UpdateExistingChore(ctx, chore, body.Assignees)
	if err != nil {
		if errors.Is(err, choreapp.ErrChoreAlreadyComplete) {
			writeJSONError(w, http.StatusConflict, "chore is no longer outstanding")
			return
		}
		choreDefinitionError(w, r, err)
		return
	}
	writeJSON(w, 200, updated)
}

func (handler *APIHandler) DeleteChore(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	choreID := chi.URLParam(r, "id")

	task, err := handler.choreService.FindChore(ctx, choreID)
	if err != nil {
		if errors.Is(err, choreapp.ErrChoreNotFound) {
			writeJSONError(w, http.StatusNotFound, "chore not found")
		} else {
			writeAPIError(w, r, http.StatusInternalServerError, "failed to load chore", err)
		}
		return
	}
	if err := handler.choreService.DeleteExistingChore(ctx, task); err != nil {
		choreDefinitionError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (handler *APIHandler) CreateRecipe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUser(ctx)

	var body recipeRequest
	if !decodeJSONBody(w, r, &body) {
		return
	}
	if body.Title == "" {
		writeJSONError(w, http.StatusBadRequest, "title is required")
		return
	}
	recipe := body.recipe()
	recipe.CreatedByUserID = user.ID
	created, err := handler.recipes.Create(ctx, recipe, body.ImageData)
	if err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "failed to create recipe", err)
		return
	}
	if created.ImageError != nil {
		slog.Error("saving recipe image via API", "method", r.Method, "route", observability.RoutePattern(r), "status", http.StatusCreated, "error", created.ImageError)
	}
	writeJSON(w, http.StatusCreated, mapRecipeResponse(created.Recipe))
}

func (handler *APIHandler) ExtractRecipe(w http.ResponseWriter, r *http.Request) {
	var body struct {
		URL string `json:"url"`
	}
	if !decodeJSONBody(w, r, &body) {
		return
	}
	if body.URL == "" {
		writeJSONError(w, http.StatusBadRequest, "url is required")
		return
	}

	recipe, err := handler.recipeExtractor.Import(r.Context(), body.URL)
	if err != nil {
		slog.Error("extracting recipe", "method", r.Method, "route", observability.RoutePattern(r), "status", http.StatusUnprocessableEntity, "error_type", fmt.Sprintf("%T", err))
		writeJSONError(w, http.StatusUnprocessableEntity, "failed to extract recipe from URL")
		return
	}

	writeJSON(w, http.StatusOK, struct {
		Title       string   `json:"title"`
		Ingredients []string `json:"ingredients"`
		Steps       []string `json:"steps"`
		PrepTime    string   `json:"prepTime,omitempty"`
		CookTime    string   `json:"cookTime,omitempty"`
		Servings    *int     `json:"servings,omitempty"`
		ImageURL    string   `json:"imageURL,omitempty"`
	}{recipe.Title, recipe.Ingredients, recipe.Steps, recipe.PrepTime, recipe.CookTime, recipe.Servings, recipe.ImageURL})
}

func (handler *APIHandler) UpdateRecipe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	recipeID := chi.URLParam(r, "id")
	existing, err := handler.recipes.PrepareUpdate(ctx, recipeID)
	if err != nil {
		if errors.Is(err, recipeapp.ErrNotFound) {
			writeJSONError(w, http.StatusNotFound, "recipe not found")
		} else {
			writeAPIError(w, r, http.StatusInternalServerError, "failed to load recipe", err)
		}
		return
	}
	var body recipeRequest
	if !decodeJSONBody(w, r, &body) {
		return
	}
	if body.Title == "" {
		writeJSONError(w, http.StatusBadRequest, "title is required")
		return
	}
	updated, err := handler.recipes.Update(ctx, existing, body.recipe(), body.ImageData)
	if err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "failed to update recipe", err)
		return
	}
	if updated.ImageError != nil {
		message := "updating recipe image via API"
		if body.ImageData != nil && *body.ImageData == "" {
			message = "clearing recipe image via API"
		}
		slog.Error(message, "method", r.Method, "route", observability.RoutePattern(r), "status", http.StatusOK, "error", updated.ImageError)
	}
	response := mapRecipeResponse(updated.Recipe)
	if updated.ReloadFallback {
		normalizeLegacyRecipeFallback(&response)
	}
	writeJSON(w, http.StatusOK, response)
}

func (handler *APIHandler) DeleteRecipe(w http.ResponseWriter, r *http.Request) {
	result, err := handler.recipes.Delete(r.Context(), chi.URLParam(r, "id"))
	if result.MealUnlinkError != nil {
		slog.Error("clearing recipe from meal plans via API", "method", r.Method, "route", observability.RoutePattern(r), "status", http.StatusNoContent, "error", result.MealUnlinkError)
	}
	if err != nil {
		if errors.Is(err, recipeapp.ErrNotFound) {
			writeJSONError(w, http.StatusNotFound, "recipe not found")
		} else if errors.Is(err, recipeapp.ErrLookup) {
			writeAPIError(w, r, http.StatusInternalServerError, "failed to load recipe", err)
		} else {
			writeAPIError(w, r, http.StatusInternalServerError, "failed to delete recipe", err)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (handler *APIHandler) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUser(ctx)

	if err := r.ParseMultipartForm(maxAvatarBytes + 1024); err != nil {
		writeJSONError(w, http.StatusBadRequest, "failed to parse form")
		return
	}

	imageBytes, contentType, ok := readUploadedImage(w, r, "avatar", maxAvatarBytes)
	if !ok {
		return
	}

	dataURI := encodeDataURI(contentType, imageBytes)
	updated, err := handler.users.SaveAvatar(ctx, user.ID, dataURI)
	if err != nil {
		if errors.Is(err, userapp.ErrReload) {
			writeAPIError(w, r, http.StatusInternalServerError, "failed to reload user", err)
		} else {
			writeAPIError(w, r, http.StatusInternalServerError, "failed to save avatar", err)
		}
		return
	}
	writeJSON(w, http.StatusOK, mapUser(updated))
}

func (handler *APIHandler) DeleteAvatar(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUser(ctx)

	if err := handler.users.RemoveAvatar(ctx, user.ID); err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "failed to remove avatar", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (handler *APIHandler) PromoteUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := chi.URLParam(r, "id")

	updated, err := handler.users.ChangeRole(ctx, userID, userapp.RoleAdmin)
	if err != nil {
		if errors.Is(err, userapp.ErrReload) {
			writeAPIError(w, r, http.StatusInternalServerError, "failed to reload user", err)
		} else {
			writeAPIError(w, r, http.StatusInternalServerError, "failed to promote user", err)
		}
		return
	}
	writeJSON(w, http.StatusOK, mapUser(updated))
}

func (handler *APIHandler) DemoteUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := chi.URLParam(r, "id")

	updated, err := handler.users.ChangeRole(ctx, userID, userapp.RoleMember)
	if err != nil {
		if errors.Is(err, userapp.ErrReload) {
			writeAPIError(w, r, http.StatusInternalServerError, "failed to reload user", err)
		} else {
			writeAPIError(w, r, http.StatusInternalServerError, "failed to demote user", err)
		}
		return
	}
	writeJSON(w, http.StatusOK, mapUser(updated))
}

func (handler *APIHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	settings, err := handler.generalSettings.GeneralSettings(ctx)
	if err != nil {
		message := "failed to load family name setting"
		if errors.Is(err, householdapp.ErrMealWeekSetting) {
			message = "failed to load meal week setting"
		}
		writeAPIError(w, r, http.StatusInternalServerError, message, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"meal_week_start": settings.MealWeekStart, "family_name": settings.FamilyName})
}

func (handler *APIHandler) PatchSettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var body map[string]json.RawMessage
	if !decodeJSONBody(w, r, &body) {
		return
	}
	updates := make(map[string]string)
	for _, key := range []string{householdapp.FamilyNameKey, householdapp.MealWeekStartKey} {
		if raw, supplied := body[key]; supplied {
			var value string
			if err := json.Unmarshal(raw, &value); err != nil || value == "" || (key == householdapp.MealWeekStartKey && !householdapp.ValidMealWeekStart(value)) {
				writeJSONError(w, http.StatusBadRequest, "invalid "+key)
				return
			}
			updates[key] = value
		}
	}
	if len(updates) == 0 {
		writeJSONError(w, http.StatusBadRequest, "at least one setting is required")
		return
	}
	if err := handler.generalSettings.UpdateGeneralSettings(ctx, updates); err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "failed to update settings", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (handler *APIHandler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUser(ctx)

	var body struct {
		Name string `json:"name"`
	}
	if !decodeJSONBody(w, r, &body) {
		return
	}
	if body.Name == "" {
		writeJSONError(w, http.StatusBadRequest, "name is required")
		return
	}

	created, err := handler.categories.Create(ctx, body.Name, user.ID)
	if err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "failed to create category", err)
		return
	}
	writeJSON(w, http.StatusCreated, mapCategory(created))
}

func (handler *APIHandler) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	categoryID := chi.URLParam(r, "id")

	var body struct {
		Name string `json:"name"`
	}
	if !decodeJSONBody(w, r, &body) {
		return
	}
	if body.Name == "" {
		writeJSONError(w, http.StatusBadRequest, "name is required")
		return
	}

	category, err := handler.categories.Update(ctx, categoryID, body.Name)
	if err != nil {
		message := "failed to update category"
		if errors.Is(err, categoryapp.ErrReload) {
			message = "failed to reload category"
		}
		writeAPIError(w, r, http.StatusInternalServerError, message, err)
		return
	}
	writeJSON(w, http.StatusOK, mapCategory(category))
}

func (handler *APIHandler) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	categoryID := chi.URLParam(r, "id")

	if err := handler.categories.Delete(ctx, categoryID); err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "failed to delete category", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (handler *APIHandler) ListTokens(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tokens, err := handler.tokens.List(ctx)
	if err != nil {
		writeAPIError(w, r, http.StatusInternalServerError, "failed to load tokens", err)
		return
	}
	if tokens == nil {
		tokens = []identityapp.TokenRecord{}
	}
	writeJSON(w, http.StatusOK, mapTokens(tokens))
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func writeAPIError(w http.ResponseWriter, r *http.Request, status int, message string, err error) {
	slog.Error("API request failed", "method", r.Method, "route", observability.RoutePattern(r), "status", status, "error", err)
	writeJSONError(w, status, message)
}

// decodeJSONBody decodes a size-limited JSON body into dest.
func decodeJSONBody(w http.ResponseWriter, r *http.Request, dest interface{}) bool {
	return decodeJSONBodyWithLimit(w, r, dest, maxJSONRequestBytes)
}

func decodeJSONBodyWithLimit(w http.ResponseWriter, r *http.Request, dest interface{}, maximumBytes int64) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maximumBytes)
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(dest); err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			writeJSONError(w, http.StatusRequestEntityTooLarge, "request body exceeds size limit")
			return false
		}
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	if err := decoder.Decode(new(interface{})); err != io.EOF {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			writeJSONError(w, http.StatusRequestEntityTooLarge, "request body exceeds size limit")
		} else {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		}
		return false
	}
	return true
}
