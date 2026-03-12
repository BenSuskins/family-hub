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

func TestAPITokenAuth_RejectsICalScopedToken(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	tokenRepo := repository.NewAPITokenRepository(database)
	userRepo := repository.NewUserRepository(database)
	ctx := context.Background()

	user, err := userRepo.Create(ctx, models.User{
		OIDCSubject: "sub-" + time.Now().String(),
		Email:       "ical-test@example.com",
		Name:        "iCal Test User",
		Role:        models.RoleAdmin,
	})
	if err != nil {
		t.Fatalf("creating test user: %v", err)
	}

	rawToken := "ical-scoped-test-token"
	_, err = tokenRepo.Create(ctx, models.APIToken{
		Name:            "iCal Token",
		TokenHash:       repository.HashToken(rawToken),
		Scope:           "ical",
		CreatedByUserID: user.ID,
	})
	if err != nil {
		t.Fatalf("creating ical token: %v", err)
	}

	apiHandler := NewAPIHandler(nil, nil, nil, nil, tokenRepo, nil, nil, nil)

	router := chi.NewRouter()
	router.Group(func(r chi.Router) {
		r.Use(middleware.APITokenAuth(tokenRepo, userRepo))
		r.Get("/api/chores", apiHandler.ListChores)
	})

	request := httptest.NewRequest(http.MethodGet, "/api/chores", nil)
	request.Header.Set("Authorization", "Bearer "+rawToken)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 for ical-scoped token on API route, got %d", recorder.Code)
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

	handler := NewAPIHandler(nil, nil, nil, nil, tokenRepo, nil, nil, nil)

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
	handler := NewAPIHandler(choreRepo, userRepo, nil, assignmentRepo, nil, choreService, nil, nil)

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
	handler := NewAPIHandler(choreRepo, userRepo, nil, assignmentRepo, nil, choreService, nil, nil)

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
	handler := NewAPIHandler(choreRepo, userRepo, nil, assignmentRepo, nil, choreService, nil, nil)

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

	handler := NewAPIHandler(nil, nil, nil, nil, nil, nil, mealPlanRepo, nil)

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

func TestListMeals_API_SnapsWeekParamToMonday(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	mealPlanRepo := repository.NewMealPlanRepository(database)
	userRepo := repository.NewUserRepository(database)
	ctx := context.Background()

	user, _ := userRepo.Create(ctx, models.User{
		OIDCSubject: "sub-meals-snap",
		Email:       "meals-snap@example.com",
		Name:        "Meals Snap User",
		Role:        models.RoleMember,
	})

	_ = mealPlanRepo.Upsert(ctx, models.MealPlan{
		Date:            "2026-03-09",
		MealType:        models.MealTypeDinner,
		Name:            "Pasta",
		CreatedByUserID: user.ID,
	})

	handler := NewAPIHandler(nil, nil, nil, nil, nil, nil, mealPlanRepo, nil)

	router := chi.NewRouter()
	router.Get("/api/meals", handler.ListMeals)

	request := httptest.NewRequest(http.MethodGet, "/api/meals?week=2026-03-11", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	var meals []models.MealPlan
	json.NewDecoder(recorder.Body).Decode(&meals)
	if len(meals) != 1 {
		t.Errorf("expected 1 meal (Monday 2026-03-09 snapped from Wednesday 2026-03-11), got %d", len(meals))
	}
}

func TestListMeals_API_InvalidWeekParam(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	mealPlanRepo := repository.NewMealPlanRepository(database)

	handler := NewAPIHandler(nil, nil, nil, nil, nil, nil, mealPlanRepo, nil)

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

	handler := NewAPIHandler(nil, nil, nil, nil, nil, nil, mealPlanRepo, nil)

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

	handler := NewAPIHandler(nil, nil, nil, nil, nil, nil, nil, recipeRepo)

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

	handler := NewAPIHandler(nil, nil, nil, nil, nil, nil, nil, recipeRepo)

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

	handler := NewAPIHandler(nil, nil, nil, nil, nil, nil, nil, recipeRepo)

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

	handler := NewAPIHandler(nil, nil, nil, nil, nil, nil, mealPlanRepo, nil)

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

	handler := NewAPIHandler(nil, nil, nil, nil, nil, nil, nil, recipeRepo)

	router := chi.NewRouter()
	router.Get("/api/recipes/{id}", handler.GetRecipe)

	request := httptest.NewRequest(http.MethodGet, "/api/recipes/nonexistent-id", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", recorder.Code, recorder.Body.String())
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

	handler := NewAPIHandler(choreRepo, userRepo, nil, nil, nil, nil, nil, nil)

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
