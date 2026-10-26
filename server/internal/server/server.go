package server

import (
	"database/sql"
	"log/slog"
	"net/http"
	"time"
	_ "time/tzdata"

	"github.com/bensuskins/family-hub/internal/config"
	"github.com/bensuskins/family-hub/internal/handlers"
	"github.com/bensuskins/family-hub/internal/httpresponse"
	"github.com/bensuskins/family-hub/internal/middleware"
	"github.com/bensuskins/family-hub/internal/observability"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/services"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
)

type Server struct {
	router *chi.Mux
	config config.Config
}

func New(database *sql.DB, cfg config.Config, senders ...services.MessageSender) *Server {
	var messageSender services.MessageSender
	if len(senders) > 0 {
		messageSender = senders[0]
	}
	userRepo := repository.NewUserRepository(database)
	categoryRepo := repository.NewCategoryRepository(database)
	choreRepo := repository.NewChoreRepository(database)
	assignmentRepo := repository.NewChoreAssignmentRepository(database)
	seriesRepo := repository.NewChoreSeriesRepository(database)
	tokenRepo := repository.NewAPITokenRepository(database)
	settingsRepo := repository.NewSettingsRepository(database)
	recipeRepo := repository.NewRecipeRepository(database)
	mealPlanRepo := repository.NewMealPlanRepository(database)
	inventoryRepo := repository.NewInventoryRepository(database)
	icalSubRepo := repository.NewICalSubscriptionRepository(database)

	choreService := services.NewChoreService(choreRepo, assignmentRepo, userRepo, seriesRepo, services.WithChoreSettings(settingsRepo))
	icalFetcher := services.NewICalFetcher(icalSubRepo)
	recipeExtractor := services.NewRecipeExtractor()
	commuteService := services.NewCommuteService(services.NewHuxleyClient(cfg.HuxleyBaseURL, cfg.HuxleyAccessToken, nil), nil)
	commuteHandler := handlers.NewCommuteHandler(commuteService)
	displayPairingHandler := handlers.NewDisplayPairingHandler(repository.NewDisplayPairingRepository(database), tokenRepo)
	displayCommuteHandler := handlers.NewDisplayCommuteHandler(repository.NewDisplayPairingRepository(database), settingsRepo, commuteService)
	scratchpadHandler := handlers.NewScratchpadHandler(repository.NewScratchpadRepository(database))
	apiHandler := handlers.NewAPIHandler(choreRepo, userRepo, categoryRepo, assignmentRepo, tokenRepo, settingsRepo, choreService, mealPlanRepo, recipeRepo, inventoryRepo, icalFetcher, recipeExtractor, cfg.OIDCUserInfoURL, cfg.OIDCClientID, cfg.OIDCIssuer)
	calendarSubscriptionsHandler := handlers.NewCalendarSubscriptionsAPIHandler(icalSubRepo, icalFetcher)
	imageHandler := handlers.NewImageHandler(recipeRepo)
	profileHandler := handlers.NewProfileHandler(userRepo)
	providerClient := services.NewSafeHTTPClient(10 * time.Second)
	weatherProvider := services.NewOpenMeteo(providerClient, "https://api.open-meteo.com/v1/forecast", time.Now)
	drivingProvider := services.NewGoogleDriving(providerClient, cfg.GoogleMapsAPIKey, "https://maps.googleapis.com/maps/api/geocode/json", "https://routes.googleapis.com/directions/v2:computeRoutes")
	departureService := services.NewDepartureService(icalFetcher, drivingProvider, time.Now)
	householdHandler := handlers.NewHouseholdHandler(settingsRepo, weatherProvider, departureService, repository.NewShoppingRepository(database))
	reminderHandler := handlers.NewChoreReminderHandler(settingsRepo, repository.NewNotificationRepository(database), cfg.PushConfigured() || messageSender != nil)
	binCollectionsHandler := handlers.NewBinCollectionsHandler(settingsRepo, cfg.PushConfigured() || messageSender != nil)
	broadcastHandler := handlers.NewBroadcastHandler(services.NewBroadcastService(repository.NewBroadcastRepository(database), repository.NewNotificationRepository(database), messageSender), messageSender != nil)
	backupHandler := handlers.NewBackupHandler(database, cfg.DatabasePath)

	router := chi.NewRouter()
	router.Use(chimiddleware.Logger)
	httpMetrics := observability.NewHTTPMetrics()
	router.Use(httpMetrics.Middleware)
	router.Use(chimiddleware.Recoverer)
	router.Use(middleware.BrowserCORS(cfg.WebOrigin))
	router.Use(chimiddleware.Compress(5))
	router.Use(middleware.SecurityHeaders)
	router.Use(httprate.LimitByIP(100, time.Minute))
	router.NotFound(func(w http.ResponseWriter, _ *http.Request) {
		httpresponse.Error(w, http.StatusNotFound, "route not found")
	})
	router.MethodNotAllowed(func(w http.ResponseWriter, _ *http.Request) {
		httpresponse.Error(w, http.StatusMethodNotAllowed, "method not allowed")
	})

	router.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	router.Get("/metrics", httpMetrics.ServeHTTP)
	router.Get("/api/client-config", apiHandler.ClientConfig)
	router.With(httprate.LimitByIP(10, time.Minute)).Post("/api/auth/exchange", apiHandler.ExchangeToken)

	router.Group(func(r chi.Router) {
		r.Use(middleware.RequireUser(tokenRepo, userRepo))

		r.Get("/avatar/{userID}", profileHandler.Serve)
		r.Get("/api/home-info", householdHandler.HomeInfo)
		r.Get("/api/weather", householdHandler.Weather)
		r.Get("/api/departures", householdHandler.Departures)
		householdHandler.RegisterShopping(r)
		r.Get("/api/me", apiHandler.Me)
		r.Post("/api/profile/avatar", apiHandler.UploadAvatar)
		r.Delete("/api/profile/avatar", apiHandler.DeleteAvatar)
		r.Get("/api/scratchpad", scratchpadHandler.Get)
		r.Put("/api/scratchpad", scratchpadHandler.Save)
		r.Get("/api/settings", apiHandler.GetSettings)
		r.Get("/api/chore-reminders", reminderHandler.Settings)
		r.Get("/api/bin-collections", binCollectionsHandler.Get)
		r.Put("/api/notification-devices/{id}", reminderHandler.Register)
		r.Delete("/api/notification-devices/{id}", reminderHandler.Remove)
		r.Get("/api/chore-definitions", apiHandler.ListChoreDefinitions)
		r.Post("/api/chore-definitions", apiHandler.CreateChoreDefinition)
		r.Get("/api/chore-definitions/{id}", apiHandler.GetChoreDefinition)
		r.Put("/api/chore-definitions/{id}", apiHandler.UpdateChoreDefinition)
		r.Delete("/api/chore-definitions/{id}", apiHandler.DeleteChoreDefinition)
		r.Get("/api/chore-history", apiHandler.ChoreHistory)
		r.Post("/api/chores/{id}/skip", apiHandler.SkipChore)
		r.Get("/api/chores", apiHandler.ListChores)
		r.Post("/api/chores", apiHandler.CreateChore)
		r.Get("/api/chores/{id}", apiHandler.GetChore)
		r.Put("/api/chores/{id}", apiHandler.UpdateChore)
		r.Delete("/api/chores/{id}", apiHandler.DeleteChore)
		r.Post("/api/chores/{id}/complete", apiHandler.CompleteChore)
		r.Get("/api/users", apiHandler.ListUsers)
		r.Get("/api/users/{id}", apiHandler.GetUser)
		r.Get("/api/categories", apiHandler.ListCategories)
		r.Get("/api/dashboard", apiHandler.DashboardStats)
		r.Get("/api/commute/trains/from/{origin}/to/{destination}", commuteHandler.ListTrains)
		r.Get("/api/meals", apiHandler.ListMeals)
		r.Post("/api/meals", apiHandler.SaveMeal)
		r.Delete("/api/meals", apiHandler.DeleteMeal)
		r.Post("/api/recipes/extract", apiHandler.ExtractRecipe)
		r.Get("/api/recipes", apiHandler.ListRecipes)
		r.Get("/api/recipes/{id}", apiHandler.GetRecipe)
		r.Post("/api/recipes", apiHandler.CreateRecipe)
		r.Put("/api/recipes/{id}", apiHandler.UpdateRecipe)
		r.Delete("/api/recipes/{id}", apiHandler.DeleteRecipe)
		r.Get("/api/recipes/{id}/image", imageHandler.ServeRecipeImage)
		r.Get("/api/calendar", apiHandler.ListCalendar)
		r.Get("/api/inventory", apiHandler.ListInventory)
		r.Post("/api/inventory/areas", apiHandler.CreateInventoryArea)
		r.Put("/api/inventory/areas/{id}", apiHandler.UpdateInventoryArea)
		r.Delete("/api/inventory/areas/{id}", apiHandler.DeleteInventoryArea)
		r.Post("/api/inventory/areas/{id}/items", apiHandler.CreateInventoryItem)
		r.Put("/api/inventory/items/{id}", apiHandler.UpdateInventoryItem)
		r.Patch("/api/inventory/items/{id}/stock", apiHandler.UpdateInventoryStock)
		r.Delete("/api/inventory/items/{id}", apiHandler.DeleteInventoryItem)

		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAdmin)

			r.Post("/api/users/{id}/promote", apiHandler.PromoteUser)
			r.Post("/api/users/{id}/demote", apiHandler.DemoteUser)
			r.Patch("/api/settings", apiHandler.PatchSettings)
			r.Patch("/api/chore-reminders", reminderHandler.Update)
			r.Post("/api/bin-collections", binCollectionsHandler.Upload)
			r.Patch("/api/bin-collections", binCollectionsHandler.Update)
			r.With(httprate.LimitByIP(5, time.Minute)).Post("/api/notifications/broadcast", broadcastHandler.Send)
			r.Patch("/api/home-info", householdHandler.UpdateHomeInfo)
			r.Post("/api/categories", apiHandler.CreateCategory)
			r.Put("/api/categories/{id}", apiHandler.UpdateCategory)
			r.Delete("/api/categories/{id}", apiHandler.DeleteCategory)
			r.Get("/api/tokens", apiHandler.ListTokens)
			r.Post("/api/tokens", apiHandler.CreateToken)
			r.Delete("/api/tokens/{id}", apiHandler.DeleteToken)
			r.Get("/api/calendars", calendarSubscriptionsHandler.List)
			r.Post("/api/calendars", calendarSubscriptionsHandler.Create)
			r.Patch("/api/calendars/{id}/color", calendarSubscriptionsHandler.UpdateColor)
			r.Post("/api/calendars/{id}/refresh", calendarSubscriptionsHandler.Refresh)
			r.Delete("/api/calendars/{id}", calendarSubscriptionsHandler.Delete)
			r.Get("/api/admin/backup", backupHandler.Backup)
			r.Post("/api/admin/restore", backupHandler.Restore)
		})
	})

	router.With(httprate.LimitByIP(5, time.Minute)).Post("/api/display-pairing/challenges", displayPairingHandler.CreateChallenge)
	router.With(httprate.LimitByIP(10, time.Minute)).Post("/api/display-pairing/challenges/{id}/claim", displayPairingHandler.Claim)
	router.Group(func(r chi.Router) {
		r.Use(middleware.RequireUser(tokenRepo, userRepo))
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAdmin)
			r.Post("/api/display-pairing/challenges/{id}/approve", displayPairingHandler.Approve)
			r.Get("/api/displays", displayPairingHandler.List)
			r.Patch("/api/displays/{id}", displayPairingHandler.UpdateSections)
			r.Patch("/api/displays/{id}/name", displayPairingHandler.UpdateName)
			r.Get("/api/displays/{id}/commute", displayPairingHandler.AdminCommutePreferences)
			r.Put("/api/displays/{id}/commute", displayPairingHandler.UpdateCommutePreferences)
			r.Delete("/api/displays/{id}", displayPairingHandler.Revoke)
		})
	})

	router.Group(func(r chi.Router) {
		r.Use(middleware.RequireDisplay(tokenRepo))
		r.Get("/api/display/settings", displayPairingHandler.Settings)
		r.Get("/api/display/home-info", householdHandler.HomeInfo)
		r.Get("/api/display/weather", householdHandler.Weather)
		r.Get("/api/display/departures", householdHandler.Departures)
		r.Get("/api/display/bin-collections", binCollectionsHandler.GetForDisplay)
		r.Get("/api/display/chores", apiHandler.ListChores)
		r.Get("/api/display/people", apiHandler.ListDisplayPeople)
		r.Post("/api/display/chores/{id}/complete", apiHandler.CompleteDisplayChore)
		r.Get("/api/display/meals", apiHandler.ListMeals)
		r.Post("/api/display/meals", apiHandler.SaveDisplayMeal)
		r.Get("/api/display/calendar", apiHandler.ListCalendar)
		r.Get("/api/display/shopping", householdHandler.ListShopping)
		r.Post("/api/display/shopping/items", householdHandler.CreateShopping)
		r.Patch("/api/display/shopping/items/{id}", householdHandler.UpdateShopping)
		r.Get("/api/display/inventory", apiHandler.ListInventory)
		r.Put("/api/display/inventory/items/{id}/stock", apiHandler.UpdateInventoryStock)
		r.Get("/api/display/commute", displayPairingHandler.CommutePreferences)
		r.Get("/api/display/trains", displayCommuteHandler.Trains)
		r.Get("/api/display/scratchpad", scratchpadHandler.Get)
		r.Put("/api/display/scratchpad", scratchpadHandler.Save)
		r.Get("/api/display/recipes", apiHandler.ListRecipes)
		r.Get("/api/display/recipes/{id}", apiHandler.GetRecipe)
	})

	return &Server{router: router, config: cfg}
}

func (server *Server) Start() error {
	address := ":" + server.config.Port
	slog.Info("starting server", "address", address)
	return http.ListenAndServe(address, server.router)
}
