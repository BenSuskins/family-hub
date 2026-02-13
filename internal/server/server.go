package server

import (
	"database/sql"
	"log/slog"
	"net/http"

	"github.com/bensuskins/family-hub/internal/config"
	"github.com/bensuskins/family-hub/internal/handlers"
	"github.com/bensuskins/family-hub/internal/middleware"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/services"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

type Server struct {
	router *chi.Mux
	config config.Config
}

func New(database *sql.DB, cfg config.Config, authService *services.AuthService) *Server {
	userRepo := repository.NewUserRepository(database)
	categoryRepo := repository.NewCategoryRepository(database)
	choreRepo := repository.NewChoreRepository(database)
	eventRepo := repository.NewEventRepository(database)
	assignmentRepo := repository.NewChoreAssignmentRepository(database)
	tokenRepo := repository.NewAPITokenRepository(database)
	settingsRepo := repository.NewSettingsRepository(database)
	recipeRepo := repository.NewRecipeRepository(database)
	mealPlanRepo := repository.NewMealPlanRepository(database)

	choreService := services.NewChoreService(choreRepo, assignmentRepo, userRepo)

	authHandler := handlers.NewAuthHandler(authService)
	dashboardHandler := handlers.NewDashboardHandler(choreRepo, eventRepo, userRepo, assignmentRepo, choreService, mealPlanRepo, categoryRepo)
	choreHandler := handlers.NewChoreHandler(choreRepo, categoryRepo, userRepo, choreService)
	eventHandler := handlers.NewEventHandler(eventRepo, categoryRepo)
	categoryHandler := handlers.NewCategoryHandler(categoryRepo)
	calendarHandler := handlers.NewCalendarHandler(choreRepo, eventRepo, userRepo, tokenRepo, mealPlanRepo, cfg.BaseURL)
	adminHandler := handlers.NewAdminHandler(userRepo, tokenRepo, settingsRepo, categoryRepo)
	apiHandler := handlers.NewAPIHandler(choreRepo, eventRepo, userRepo, categoryRepo, assignmentRepo, tokenRepo)
	icalHandler := handlers.NewICalHandler(choreRepo, eventRepo, userRepo, tokenRepo, settingsRepo, mealPlanRepo, cfg.HAAPIToken)
	haHandler := handlers.NewHASensorHandler(choreRepo, userRepo, cfg.HAAPIToken)
	recipeHandler := handlers.NewRecipeHandler(recipeRepo, categoryRepo, mealPlanRepo)
	mealHandler := handlers.NewMealHandler(mealPlanRepo, recipeRepo)

	router := chi.NewRouter()

	router.Use(chimiddleware.Logger)
	router.Use(chimiddleware.Recoverer)
	router.Use(chimiddleware.Compress(5))
	router.Use(middleware.InjectFamilyName(settingsRepo))

	router.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	router.Get("/login", authHandler.LoginPage)
	router.Get("/auth/callback", authHandler.Callback)
	router.Get("/logout", authHandler.Logout)

	router.Get("/ical", icalHandler.Feed)
	router.Get("/api/ha/sensors", haHandler.Sensors)

	router.Group(func(r chi.Router) {
		r.Use(middleware.RequireAuth(authService))

		r.Get("/", dashboardHandler.Dashboard)
		r.Get("/leaderboard", dashboardHandler.Leaderboard)

		r.Get("/chores", choreHandler.List)
		r.Get("/chores/{id}/detail", choreHandler.Detail)
		r.Post("/chores/{id}/complete", choreHandler.Complete)

		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAdmin)

			r.Get("/chores/new", choreHandler.CreateForm)
			r.Post("/chores", choreHandler.Create)
			r.Get("/chores/{id}/edit", choreHandler.EditForm)
			r.Post("/chores/{id}", choreHandler.Update)
			r.Post("/chores/{id}/delete", choreHandler.Delete)
		})

		r.Get("/events", eventHandler.List)
		r.Get("/events/{id}/detail", eventHandler.Detail)
		r.Get("/events/new", eventHandler.CreateForm)
		r.Post("/events", eventHandler.Create)

		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAdmin)

			r.Get("/events/{id}/edit", eventHandler.EditForm)
			r.Post("/events/{id}", eventHandler.Update)
			r.Post("/events/{id}/delete", eventHandler.Delete)
		})

		r.Get("/meals", mealHandler.Planner)
		r.Post("/meals", mealHandler.SaveMeal)
		r.Post("/meals/delete", mealHandler.DeleteMeal)
		r.Get("/meals/cell", mealHandler.Cell)

		r.Get("/recipes", recipeHandler.List)
		r.Get("/recipes/new", recipeHandler.CreateForm)
		r.Get("/recipes/ingredient-group", recipeHandler.IngredientGroup)
		r.Get("/recipes/{id}", recipeHandler.Detail)
		r.Post("/recipes", recipeHandler.Create)
		r.Get("/recipes/{id}/edit", recipeHandler.EditForm)
		r.Post("/recipes/{id}", recipeHandler.Update)
		r.Post("/recipes/{id}/delete", recipeHandler.Delete)

		r.Get("/calendar", calendarHandler.Calendar)
		r.Post("/calendar/share", calendarHandler.ShareInfo)

		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAdmin)

			r.Post("/categories", categoryHandler.Create)
			r.Post("/categories/{id}", categoryHandler.Update)
			r.Post("/categories/{id}/delete", categoryHandler.Delete)

			r.Get("/admin/users", adminHandler.Users)
			r.Post("/admin/users/{id}/promote", adminHandler.PromoteUser)
			r.Post("/admin/users/{id}/demote", adminHandler.DemoteUser)
			r.Post("/admin/settings", adminHandler.UpdateSettings)
		})
	})

	router.Group(func(r chi.Router) {
		r.Use(middleware.APITokenAuth(tokenRepo, userRepo))

		r.Get("/api/chores", apiHandler.ListChores)
		r.Get("/api/chores/{id}", apiHandler.GetChore)
		r.Get("/api/events", apiHandler.ListEvents)
		r.Get("/api/events/{id}", apiHandler.GetEvent)
		r.Get("/api/users", apiHandler.ListUsers)
		r.Get("/api/users/{id}", apiHandler.GetUser)
		r.Get("/api/categories", apiHandler.ListCategories)
		r.Get("/api/dashboard", apiHandler.DashboardStats)
	})

	router.Group(func(r chi.Router) {
		r.Use(middleware.RequireAuth(authService))
		r.Use(middleware.RequireAdmin)

		r.Post("/api/tokens", apiHandler.CreateToken)
		r.Delete("/api/tokens/{id}", apiHandler.DeleteToken)
	})

	server := &Server{
		router: router,
		config: cfg,
	}

	return server
}

func (server *Server) Start() error {
	address := ":" + server.config.Port
	slog.Info("starting server", "address", address)
	return http.ListenAndServe(address, server.router)
}
