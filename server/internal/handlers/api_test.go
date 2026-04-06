package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bensuskins/family-hub/internal/middleware"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/services"
	"github.com/bensuskins/family-hub/internal/testutil"
	"github.com/go-chi/chi/v5"
)

func TestRequireUser_RejectsNonAPIScopedToken(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	tokenRepo := repository.NewAPITokenRepository(database)
	userRepo := repository.NewUserRepository(database)
	ctx := context.Background()

	user, err := userRepo.Create(ctx, models.User{
		OIDCSubject: "sub-" + time.Now().String(),
		Email:       "scope-test@example.com",
		Name:        "Scope Test User",
		Role:        models.RoleAdmin,
	})
	if err != nil {
		t.Fatalf("creating test user: %v", err)
	}

	rawToken := "non-api-scoped-test-token"
	_, err = tokenRepo.Create(ctx, models.APIToken{
		Name:            "Stale Token",
		TokenHash:       repository.HashToken(rawToken),
		Scope:           "ical", // legacy value from before scope removal
		CreatedByUserID: user.ID,
	})
	if err != nil {
		t.Fatalf("creating stale-scope token: %v", err)
	}

	apiHandler := NewAPIHandler(nil, nil, nil, nil, tokenRepo, nil, nil, nil, nil, nil, "", "", "")

	router := chi.NewRouter()
	router.Group(func(r chi.Router) {
		r.Use(middleware.RequireUser(nil, tokenRepo, userRepo))
		r.Get("/api/chores", apiHandler.ListChores)
	})

	request := httptest.NewRequest(http.MethodGet, "/api/chores", nil)
	request.Header.Set("Authorization", "Bearer "+rawToken)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 for non-api-scoped token on API route, got %d", recorder.Code)
	}
}

func TestDeleteToken(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	tokenRepo := repository.NewAPITokenRepository(database)
	userRepo := repository.NewUserRepository(database)
	ctx := context.Background()

	user, err := userRepo.Create(ctx, models.User{
		OIDCSubject: "sub-" + time.Now().String(),
		Email:       "test@example.com",
		Name:        "Test User",
		Role:        models.RoleAdmin,
	})
	if err != nil {
		t.Fatalf("creating test user: %v", err)
	}

	created, err := tokenRepo.Create(ctx, models.APIToken{
		Name:            "To Revoke",
		TokenHash:       "hash-revoke",
		CreatedByUserID: user.ID,
	})
	if err != nil {
		t.Fatalf("creating token: %v", err)
	}

	handler := NewAPIHandler(nil, nil, nil, nil, tokenRepo, nil, nil, nil, nil, nil, "", "", "")

	router := chi.NewRouter()
	router.Delete("/api/tokens/{id}", handler.DeleteToken)

	request := httptest.NewRequest(http.MethodDelete, "/api/tokens/"+created.ID, nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	tokens, err := tokenRepo.FindAll(ctx)
	if err != nil {
		t.Fatalf("listing tokens after delete: %v", err)
	}
	if len(tokens) != 0 {
		t.Errorf("expected 0 tokens after revoke, got %d", len(tokens))
	}
}

func TestCompleteChore_API(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	choreRepo := repository.NewChoreRepository(database)
	userRepo := repository.NewUserRepository(database)
	assignmentRepo := repository.NewChoreAssignmentRepository(database)
	ctx := context.Background()

	user, _ := userRepo.Create(ctx, models.User{
		OIDCSubject: "sub-complete",
		Email:       "complete@example.com",
		Name:        "Complete User",
		Role:        models.RoleMember,
	})

	chore, _ := choreRepo.Create(ctx, models.Chore{
		Name:            "Chore to complete",
		CreatedByUserID: user.ID,
		Status:          models.ChoreStatusPending,
	})

	choreService := services.NewChoreService(choreRepo, assignmentRepo, userRepo)
	handler := NewAPIHandler(choreRepo, userRepo, nil, assignmentRepo, nil, choreService, nil, nil, nil, nil, "", "", "")

	router := chi.NewRouter()
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), middleware.UserContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	router.Post("/api/chores/{id}/complete", handler.CompleteChore)

	request := httptest.NewRequest(http.MethodPost, "/api/chores/"+chore.ID+"/complete", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", recorder.Code, recorder.Body.String())
	}

	updated, _ := choreRepo.FindByID(ctx, chore.ID)
	if updated.Status != models.ChoreStatusCompleted {
		t.Errorf("expected chore to be completed, got %s", updated.Status)
	}
}

func TestCompleteChore_API_NotFound(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	choreRepo := repository.NewChoreRepository(database)
	userRepo := repository.NewUserRepository(database)
	assignmentRepo := repository.NewChoreAssignmentRepository(database)
	ctx := context.Background()

	user, _ := userRepo.Create(ctx, models.User{
		OIDCSubject: "sub-notfound",
		Email:       "notfound@example.com",
		Name:        "Not Found User",
		Role:        models.RoleMember,
	})

	choreService := services.NewChoreService(choreRepo, assignmentRepo, userRepo)
	handler := NewAPIHandler(choreRepo, userRepo, nil, assignmentRepo, nil, choreService, nil, nil, nil, nil, "", "", "")

	router := chi.NewRouter()
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), middleware.UserContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	router.Post("/api/chores/{id}/complete", handler.CompleteChore)

	request := httptest.NewRequest(http.MethodPost, "/api/chores/nonexistent-id/complete", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestCompleteChore_API_AlreadyComplete(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	choreRepo := repository.NewChoreRepository(database)
	userRepo := repository.NewUserRepository(database)
	assignmentRepo := repository.NewChoreAssignmentRepository(database)
	ctx := context.Background()

	user, _ := userRepo.Create(ctx, models.User{
		OIDCSubject: "sub-alreadycomplete",
		Email:       "alreadycomplete@example.com",
		Name:        "Already Complete User",
		Role:        models.RoleMember,
	})

	chore, _ := choreRepo.Create(ctx, models.Chore{
		Name:            "Already completed chore",
		CreatedByUserID: user.ID,
		Status:          models.ChoreStatusCompleted,
	})

	choreService := services.NewChoreService(choreRepo, assignmentRepo, userRepo)
	handler := NewAPIHandler(choreRepo, userRepo, nil, assignmentRepo, nil, choreService, nil, nil, nil, nil, "", "", "")

	router := chi.NewRouter()
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), middleware.UserContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	router.Post("/api/chores/{id}/complete", handler.CompleteChore)

	request := httptest.NewRequest(http.MethodPost, "/api/chores/"+chore.ID+"/complete", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestListMeals_API(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	mealPlanRepo := repository.NewMealPlanRepository(database)
	userRepo := repository.NewUserRepository(database)
	ctx := context.Background()

	user, _ := userRepo.Create(ctx, models.User{
		OIDCSubject: "sub-meals",
		Email:       "meals@example.com",
		Name:        "Meals User",
		Role:        models.RoleMember,
	})

	_ = mealPlanRepo.Upsert(ctx, models.MealPlan{
		Date:            "2026-03-09",
		MealType:        models.MealTypeDinner,
		Name:            "Pasta",
		CreatedByUserID: user.ID,
	})

	handler := NewAPIHandler(nil, nil, nil, nil, nil, nil, mealPlanRepo, nil, nil, nil, "", "", "")

	router := chi.NewRouter()
	router.Get("/api/meals", handler.ListMeals)

	request := httptest.NewRequest(http.MethodGet, "/api/meals?week=2026-03-09", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	var meals []models.MealPlan
	json.NewDecoder(recorder.Body).Decode(&meals)
	if len(meals) != 1 {
		t.Errorf("expected 1 meal, got %d", len(meals))
	}
}

func TestListMeals_API_UsesWeekParamAsStartDate(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	mealPlanRepo := repository.NewMealPlanRepository(database)
	userRepo := repository.NewUserRepository(database)
	ctx := context.Background()

	user, _ := userRepo.Create(ctx, models.User{
		OIDCSubject: "sub-meals-start",
		Email:       "meals-start@example.com",
		Name:        "Meals Start User",
		Role:        models.RoleMember,
	})

	_ = mealPlanRepo.Upsert(ctx, models.MealPlan{
		Date:            "2026-03-13",
		MealType:        models.MealTypeDinner,
		Name:            "Friday Pasta",
		CreatedByUserID: user.ID,
	})
	_ = mealPlanRepo.Upsert(ctx, models.MealPlan{
		Date:            "2026-03-19",
		MealType:        models.MealTypeDinner,
		Name:            "Thursday Pasta",
		CreatedByUserID: user.ID,
	})
	_ = mealPlanRepo.Upsert(ctx, models.MealPlan{
		Date:            "2026-03-20",
		MealType:        models.MealTypeDinner,
		Name:            "Next Friday Pasta",
		CreatedByUserID: user.ID,
	})

	handler := NewAPIHandler(nil, nil, nil, nil, nil, nil, mealPlanRepo, nil, nil, nil, "", "", "")

	router := chi.NewRouter()
	router.Get("/api/meals", handler.ListMeals)

	request := httptest.NewRequest(http.MethodGet, "/api/meals?week=2026-03-13", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	var meals []models.MealPlan
	json.NewDecoder(recorder.Body).Decode(&meals)
	if len(meals) != 2 {
		t.Errorf("expected 2 meals (Fri 2026-03-13 through Thu 2026-03-19), got %d", len(meals))
	}
}

func TestListMeals_API_InvalidWeekParam(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	mealPlanRepo := repository.NewMealPlanRepository(database)

	handler := NewAPIHandler(nil, nil, nil, nil, nil, nil, mealPlanRepo, nil, nil, nil, "", "", "")

	router := chi.NewRouter()
	router.Get("/api/meals", handler.ListMeals)

	request := httptest.NewRequest(http.MethodGet, "/api/meals?week=not-a-date", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", recorder.Code)
	}
}

func TestListMeals_API_DefaultsToCurrentWeek(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	mealPlanRepo := repository.NewMealPlanRepository(database)

	handler := NewAPIHandler(nil, nil, nil, nil, nil, nil, mealPlanRepo, nil, nil, nil, "", "", "")

	router := chi.NewRouter()
	router.Get("/api/meals", handler.ListMeals)

	request := httptest.NewRequest(http.MethodGet, "/api/meals", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", recorder.Code)
	}
}

func TestListRecipes_API(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	recipeRepo := repository.NewRecipeRepository(database)
	userRepo := repository.NewUserRepository(database)
	ctx := context.Background()

	user, _ := userRepo.Create(ctx, models.User{
		OIDCSubject: "sub-recipes",
		Email:       "recipes@example.com",
		Name:        "Recipe User",
		Role:        models.RoleMember,
	})

	_, _ = recipeRepo.Create(ctx, models.Recipe{
		Title:           "Pasta Bake",
		CreatedByUserID: user.ID,
	})

	handler := NewAPIHandler(nil, nil, nil, nil, nil, nil, nil, recipeRepo, nil, nil, "", "", "")

	router := chi.NewRouter()
	router.Get("/api/recipes", handler.ListRecipes)

	request := httptest.NewRequest(http.MethodGet, "/api/recipes", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	var recipes []models.Recipe
	json.NewDecoder(recorder.Body).Decode(&recipes)
	if len(recipes) != 1 {
		t.Errorf("expected 1 recipe, got %d", len(recipes))
	}
}

func TestGetRecipe_API(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	recipeRepo := repository.NewRecipeRepository(database)
	userRepo := repository.NewUserRepository(database)
	ctx := context.Background()

	user, _ := userRepo.Create(ctx, models.User{
		OIDCSubject: "sub-recipe-detail",
		Email:       "detail@example.com",
		Name:        "Detail User",
		Role:        models.RoleMember,
	})

	created, _ := recipeRepo.Create(ctx, models.Recipe{
		Title:           "Fish Pie",
		CreatedByUserID: user.ID,
	})

	handler := NewAPIHandler(nil, nil, nil, nil, nil, nil, nil, recipeRepo, nil, nil, "", "", "")

	router := chi.NewRouter()
	router.Get("/api/recipes/{id}", handler.GetRecipe)

	request := httptest.NewRequest(http.MethodGet, "/api/recipes/"+created.ID, nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	var recipe models.Recipe
	json.NewDecoder(recorder.Body).Decode(&recipe)
	if recipe.Title != "Fish Pie" {
		t.Errorf("expected title Fish Pie, got %s", recipe.Title)
	}
}

func TestListRecipes_API_Empty(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	recipeRepo := repository.NewRecipeRepository(database)

	handler := NewAPIHandler(nil, nil, nil, nil, nil, nil, nil, recipeRepo, nil, nil, "", "", "")

	router := chi.NewRouter()
	router.Get("/api/recipes", handler.ListRecipes)

	request := httptest.NewRequest(http.MethodGet, "/api/recipes", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	body := strings.TrimSpace(recorder.Body.String())
	if body != "[]" {
		t.Errorf("expected empty JSON array [], got %s", body)
	}
}

func TestListMeals_API_Empty(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	mealPlanRepo := repository.NewMealPlanRepository(database)

	handler := NewAPIHandler(nil, nil, nil, nil, nil, nil, mealPlanRepo, nil, nil, nil, "", "", "")

	router := chi.NewRouter()
	router.Get("/api/meals", handler.ListMeals)

	request := httptest.NewRequest(http.MethodGet, "/api/meals?week=2026-03-09", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	body := strings.TrimSpace(recorder.Body.String())
	if body != "[]" {
		t.Errorf("expected empty JSON array [], got %s", body)
	}
}

func TestGetRecipe_API_NotFound(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	recipeRepo := repository.NewRecipeRepository(database)

	handler := NewAPIHandler(nil, nil, nil, nil, nil, nil, nil, recipeRepo, nil, nil, "", "", "")

	router := chi.NewRouter()
	router.Get("/api/recipes/{id}", handler.GetRecipe)

	request := httptest.NewRequest(http.MethodGet, "/api/recipes/nonexistent-id", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestListCalendar_API(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	choreRepo := repository.NewChoreRepository(database)
	userRepo := repository.NewUserRepository(database)
	ctx := context.Background()

	user, _ := userRepo.Create(ctx, models.User{
		OIDCSubject: "sub-calendar",
		Email:       "calendar@example.com",
		Name:        "Calendar User",
		Role:        models.RoleMember,
	})

	dueDate, _ := time.Parse("2006-01-02", "2026-03-15")
	_, _ = choreRepo.Create(ctx, models.Chore{
		Name:            "March chore",
		CreatedByUserID: user.ID,
		DueDate:         &dueDate,
		Status:          models.ChoreStatusPending,
	})

	handler := NewAPIHandler(choreRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, "", "", "")

	router := chi.NewRouter()
	router.Get("/api/calendar", handler.ListCalendar)

	request := httptest.NewRequest(http.MethodGet, "/api/calendar?month=2026-03", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	var body map[string]interface{}
	json.NewDecoder(recorder.Body).Decode(&body)

	chores, ok := body["chores"]
	if !ok {
		t.Fatal("expected chores key in response")
	}
	if len(chores.([]interface{})) != 1 {
		t.Errorf("expected 1 chore, got %d", len(chores.([]interface{})))
	}
}

func TestListCalendar_API_InvalidMonthParam(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	choreRepo := repository.NewChoreRepository(database)

	handler := NewAPIHandler(choreRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, "", "", "")

	router := chi.NewRouter()
	router.Get("/api/calendar", handler.ListCalendar)

	request := httptest.NewRequest(http.MethodGet, "/api/calendar?month=not-a-month", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", recorder.Code)
	}
}

func TestListCalendar_API_DefaultsToCurrentMonth(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	choreRepo := repository.NewChoreRepository(database)

	handler := NewAPIHandler(choreRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, "", "", "")

	router := chi.NewRouter()
	router.Get("/api/calendar", handler.ListCalendar)

	request := httptest.NewRequest(http.MethodGet, "/api/calendar", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", recorder.Code)
	}
}

func TestListCalendar_API_EmptyResult(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	choreRepo := repository.NewChoreRepository(database)

	handler := NewAPIHandler(choreRepo, nil, nil, nil, nil, nil, nil, nil, nil, nil, "", "", "")

	router := chi.NewRouter()
	router.Get("/api/calendar", handler.ListCalendar)

	request := httptest.NewRequest(http.MethodGet, "/api/calendar?month=2020-01", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	var body map[string]interface{}
	json.NewDecoder(recorder.Body).Decode(&body)

	chores, ok := body["chores"]
	if !ok {
		t.Fatal("expected chores key in response")
	}
	if chores == nil {
		t.Error("expected chores to be [] not null")
	}
	if len(chores.([]interface{})) != 0 {
		t.Errorf("expected 0 chores, got %d", len(chores.([]interface{})))
	}
}

func TestDashboardStats_EmptyListsAreNotNull(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	choreRepo := repository.NewChoreRepository(database)
	userRepo := repository.NewUserRepository(database)

	handler := NewAPIHandler(choreRepo, userRepo, nil, nil, nil, nil, nil, nil, nil, nil, "", "", "")

	router := chi.NewRouter()
	router.Get("/api/dashboard", handler.DashboardStats)

	request := httptest.NewRequest(http.MethodGet, "/api/dashboard", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	var body map[string]json.RawMessage
	json.NewDecoder(recorder.Body).Decode(&body)

	if string(body["chores_due_today_list"]) == "null" {
		t.Error("chores_due_today_list must be [] not null — iOS non-optional [Chore] cannot decode null")
	}
	if string(body["chores_overdue_list"]) == "null" {
		t.Error("chores_overdue_list must be [] not null — iOS non-optional [Chore] cannot decode null")
	}
}

func TestListRecipes_NilIngredientItemsAreEmptyArray(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	recipeRepo := repository.NewRecipeRepository(database)
	userRepo := repository.NewUserRepository(database)
	ctx := context.Background()

	user, _ := userRepo.Create(ctx, models.User{
		OIDCSubject: "sub-recipe-items",
		Email:       "items@example.com",
		Name:        "Items User",
		Role:        models.RoleMember,
	})

	_, _ = recipeRepo.Create(ctx, models.Recipe{
		Title:           "Recipe With Group",
		CreatedByUserID: user.ID,
		Ingredients: []models.IngredientGroup{
			{Name: "Main", Items: nil},
		},
	})

	handler := NewAPIHandler(nil, nil, nil, nil, nil, nil, nil, recipeRepo, nil, nil, "", "", "")

	router := chi.NewRouter()
	router.Get("/api/recipes", handler.ListRecipes)

	request := httptest.NewRequest(http.MethodGet, "/api/recipes", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	body := recorder.Body.String()
	if strings.Contains(body, `"items":null`) {
		t.Error(`ingredient items must be [] not null — iOS non-optional [String] cannot decode null`)
	}
}

func TestAPIHandler_ClientConfig(t *testing.T) {
	tests := []struct {
		name         string
		clientID     string
		oidcIssuer   string
		wantStatus   int
		wantClientID string
		wantIssuer   string
	}{
		{
			name:         "returns client config as JSON",
			clientID:     "familyhub-ios",
			oidcIssuer:   "https://auth.example.com/application/o/familyhub",
			wantStatus:   http.StatusOK,
			wantClientID: "familyhub-ios",
			wantIssuer:   "https://auth.example.com/application/o/familyhub",
		},
		{
			name:       "empty config still returns 200",
			clientID:   "",
			oidcIssuer: "",
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewAPIHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, "", tt.clientID, tt.oidcIssuer)

			request := httptest.NewRequest(http.MethodGet, "/api/client-config", nil)
			recorder := httptest.NewRecorder()
			handler.ClientConfig(recorder, request)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("want status %d, got %d", tt.wantStatus, recorder.Code)
			}

			var body struct {
				ClientID string `json:"clientID"`
				Issuer   string `json:"issuer"`
			}
			if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
				t.Fatalf("decoding response: %v", err)
			}
			if body.ClientID != tt.wantClientID {
				t.Errorf("want clientID %q, got %q", tt.wantClientID, body.ClientID)
			}
			if body.Issuer != tt.wantIssuer {
				t.Errorf("want issuer %q, got %q", tt.wantIssuer, body.Issuer)
			}
		})
	}
}

func TestDashboardStats_IncludesChores(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	choreRepo := repository.NewChoreRepository(database)
	userRepo := repository.NewUserRepository(database)
	ctx := context.Background()

	user, _ := userRepo.Create(ctx, models.User{
		OIDCSubject: "sub-dashboard",
		Email:       "dashboard@example.com",
		Name:        "Dashboard User",
		Role:        models.RoleMember,
	})

	today := time.Now().Truncate(24 * time.Hour)
	_, _ = choreRepo.Create(ctx, models.Chore{
		Name:            "Due today chore",
		CreatedByUserID: user.ID,
		DueDate:         &today,
		Status:          models.ChoreStatusPending,
	})

	yesterday := today.AddDate(0, 0, -1)
	_, _ = choreRepo.Create(ctx, models.Chore{
		Name:            "Overdue chore",
		CreatedByUserID: user.ID,
		DueDate:         &yesterday,
		Status:          models.ChoreStatusOverdue,
	})

	handler := NewAPIHandler(choreRepo, userRepo, nil, nil, nil, nil, nil, nil, nil, nil, "", "", "")

	router := chi.NewRouter()
	router.Get("/api/dashboard", handler.DashboardStats)

	request := httptest.NewRequest(http.MethodGet, "/api/dashboard", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	var body map[string]interface{}
	json.NewDecoder(recorder.Body).Decode(&body)

	dueTodayList, ok := body["chores_due_today_list"]
	if !ok {
		t.Fatal("expected chores_due_today_list in response")
	}
	if len(dueTodayList.([]interface{})) != 1 {
		t.Errorf("expected 1 chore due today, got %d", len(dueTodayList.([]interface{})))
	}

	overdueList, ok := body["chores_overdue_list"]
	if !ok {
		t.Fatal("expected chores_overdue_list in response")
	}
	if len(overdueList.([]interface{})) != 1 {
		t.Errorf("expected 1 overdue chore, got %d", len(overdueList.([]interface{})))
	}

	if count, ok := body["chores_due_today"]; !ok {
		t.Fatal("expected chores_due_today count in response")
	} else if int(count.(float64)) != 1 {
		t.Errorf("expected chores_due_today count 1, got %v", count)
	}

	if count, ok := body["chores_overdue"]; !ok {
		t.Fatal("expected chores_overdue count in response")
	} else if int(count.(float64)) != 1 {
		t.Errorf("expected chores_overdue count 1, got %v", count)
	}
}

func TestCreateRecipe_API(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	recipeRepo := repository.NewRecipeRepository(database)
	userRepo := repository.NewUserRepository(database)
	ctx := context.Background()

	user, _ := userRepo.Create(ctx, models.User{
		OIDCSubject: "sub-create-recipe",
		Email:       "create@example.com",
		Name:        "Creator",
		Role:        models.RoleMember,
	})

	handler := NewAPIHandler(nil, nil, nil, nil, nil, nil, nil, recipeRepo, nil, nil, "", "", "")

	router := chi.NewRouter()
	router.Post("/api/recipes", func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), middleware.UserContextKey, user)
		handler.CreateRecipe(w, r.WithContext(ctx))
	})

	body := `{"title":"Pasta Carbonara","steps":["Boil pasta","Mix eggs"],"ingredients":[{"name":"Main","items":["pasta","eggs"]}],"mealType":"dinner","servings":4,"prepTime":"10 min","cookTime":"20 min","sourceURL":"https://example.com"}`
	request := httptest.NewRequest(http.MethodPost, "/api/recipes", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", recorder.Code, recorder.Body.String())
	}

	var recipe models.Recipe
	json.NewDecoder(recorder.Body).Decode(&recipe)
	if recipe.Title != "Pasta Carbonara" {
		t.Errorf("expected title Pasta Carbonara, got %s", recipe.Title)
	}
	if recipe.MealType == nil || string(*recipe.MealType) != "dinner" {
		t.Errorf("expected mealType dinner, got %v", recipe.MealType)
	}
	if recipe.Servings == nil || *recipe.Servings != 4 {
		t.Errorf("expected servings 4, got %v", recipe.Servings)
	}
	if len(recipe.Steps) != 2 {
		t.Errorf("expected 2 steps, got %d", len(recipe.Steps))
	}
}

func TestCreateRecipe_API_MissingTitle(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	recipeRepo := repository.NewRecipeRepository(database)
	userRepo := repository.NewUserRepository(database)
	ctx := context.Background()

	user, _ := userRepo.Create(ctx, models.User{
		OIDCSubject: "sub-create-recipe-notitle",
		Email:       "notitle@example.com",
		Name:        "No Title",
		Role:        models.RoleMember,
	})

	handler := NewAPIHandler(nil, nil, nil, nil, nil, nil, nil, recipeRepo, nil, nil, "", "", "")

	router := chi.NewRouter()
	router.Post("/api/recipes", func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), middleware.UserContextKey, user)
		handler.CreateRecipe(w, r.WithContext(ctx))
	})

	body := `{"steps":["step 1"]}`
	request := httptest.NewRequest(http.MethodPost, "/api/recipes", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing title, got %d", recorder.Code)
	}
}

func TestCreateRecipe_API_WithImage(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	recipeRepo := repository.NewRecipeRepository(database)
	userRepo := repository.NewUserRepository(database)
	ctx := context.Background()

	user, _ := userRepo.Create(ctx, models.User{
		OIDCSubject: "sub-create-recipe-img",
		Email:       "img@example.com",
		Name:        "Img User",
		Role:        models.RoleMember,
	})

	handler := NewAPIHandler(nil, nil, nil, nil, nil, nil, nil, recipeRepo, nil, nil, "", "", "")

	router := chi.NewRouter()
	router.Post("/api/recipes", func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), middleware.UserContextKey, user)
		handler.CreateRecipe(w, r.WithContext(ctx))
	})

	body := `{"title":"Photo Recipe","imageData":"data:image/png;base64,iVBORw0KGgo="}`
	request := httptest.NewRequest(http.MethodPost, "/api/recipes", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", recorder.Code, recorder.Body.String())
	}

	var recipe models.Recipe
	json.NewDecoder(recorder.Body).Decode(&recipe)
	if !recipe.HasImage {
		t.Error("expected HasImage to be true after uploading image data")
	}

	imageData, _ := recipeRepo.FindImageData(ctx, recipe.ID)
	if imageData != "data:image/png;base64,iVBORw0KGgo=" {
		t.Errorf("expected stored image data URI, got %q", imageData)
	}
}

func TestUpdateRecipe_API(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	recipeRepo := repository.NewRecipeRepository(database)
	userRepo := repository.NewUserRepository(database)
	ctx := context.Background()

	user, _ := userRepo.Create(ctx, models.User{
		OIDCSubject: "sub-update-recipe",
		Email:       "update@example.com",
		Name:        "Updater",
		Role:        models.RoleMember,
	})

	created, _ := recipeRepo.Create(ctx, models.Recipe{
		Title:           "Old Title",
		CreatedByUserID: user.ID,
	})

	handler := NewAPIHandler(nil, nil, nil, nil, nil, nil, nil, recipeRepo, nil, nil, "", "", "")

	router := chi.NewRouter()
	router.Put("/api/recipes/{id}", handler.UpdateRecipe)

	body := `{"title":"New Title","steps":["step 1"],"servings":2,"prepTime":"5 min"}`
	request := httptest.NewRequest(http.MethodPut, "/api/recipes/"+created.ID, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	var recipe models.Recipe
	json.NewDecoder(recorder.Body).Decode(&recipe)
	if recipe.Title != "New Title" {
		t.Errorf("expected title New Title, got %s", recipe.Title)
	}
	if recipe.Servings == nil || *recipe.Servings != 2 {
		t.Errorf("expected servings 2, got %v", recipe.Servings)
	}
}

func TestUpdateRecipe_API_NotFound(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	recipeRepo := repository.NewRecipeRepository(database)

	handler := NewAPIHandler(nil, nil, nil, nil, nil, nil, nil, recipeRepo, nil, nil, "", "", "")

	router := chi.NewRouter()
	router.Put("/api/recipes/{id}", handler.UpdateRecipe)

	body := `{"title":"Nope"}`
	request := httptest.NewRequest(http.MethodPut, "/api/recipes/nonexistent", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", recorder.Code)
	}
}

func TestUpdateRecipe_API_ImageHandling(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	recipeRepo := repository.NewRecipeRepository(database)
	userRepo := repository.NewUserRepository(database)
	ctx := context.Background()

	user, _ := userRepo.Create(ctx, models.User{
		OIDCSubject: "sub-update-recipe-img",
		Email:       "updimg@example.com",
		Name:        "Img Updater",
		Role:        models.RoleMember,
	})

	created, _ := recipeRepo.Create(ctx, models.Recipe{
		Title:           "Img Recipe",
		CreatedByUserID: user.ID,
	})

	handler := NewAPIHandler(nil, nil, nil, nil, nil, nil, nil, recipeRepo, nil, nil, "", "", "")

	router := chi.NewRouter()
	router.Put("/api/recipes/{id}", handler.UpdateRecipe)

	// Add image
	body := `{"title":"Img Recipe","imageData":"data:image/png;base64,iVBORw0KGgo="}`
	request := httptest.NewRequest(http.MethodPut, "/api/recipes/"+created.ID, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	var recipe models.Recipe
	json.NewDecoder(recorder.Body).Decode(&recipe)
	if !recipe.HasImage {
		t.Error("expected HasImage true after setting image")
	}

	// Clear image with empty string
	body = `{"title":"Img Recipe","imageData":""}`
	request = httptest.NewRequest(http.MethodPut, "/api/recipes/"+created.ID, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	json.NewDecoder(recorder.Body).Decode(&recipe)
	if recipe.HasImage {
		t.Error("expected HasImage false after clearing image")
	}
}
