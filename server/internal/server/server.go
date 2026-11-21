package server

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"time"
	_ "time/tzdata"

	binsapp "github.com/bensuskins/family-hub/internal/application/bins"
	categoryapp "github.com/bensuskins/family-hub/internal/application/categories"
	dashboardapp "github.com/bensuskins/family-hub/internal/application/dashboard"

	mealapp "github.com/bensuskins/family-hub/internal/application/meals"

	identityapp "github.com/bensuskins/family-hub/internal/application/identity"
	userapp "github.com/bensuskins/family-hub/internal/application/users"

	displayapp "github.com/bensuskins/family-hub/internal/application/displays"
	"github.com/google/uuid"

	notifications "github.com/bensuskins/family-hub/internal/application/notifications"

	recipeapp "github.com/bensuskins/family-hub/internal/application/recipes"

	calendarapp "github.com/bensuskins/family-hub/internal/application/calendar"
	commuteapp "github.com/bensuskins/family-hub/internal/application/commute"
	weatherapp "github.com/bensuskins/family-hub/internal/application/weather"

	"github.com/bensuskins/family-hub/internal/apidocs"
	choreapp "github.com/bensuskins/family-hub/internal/application/chores"
	householdapp "github.com/bensuskins/family-hub/internal/application/household"
	inventoryapp "github.com/bensuskins/family-hub/internal/application/inventory"
	scratchapp "github.com/bensuskins/family-hub/internal/application/scratchpad"
	shoppingapp "github.com/bensuskins/family-hub/internal/application/shopping"
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
	router     *chi.Mux
	httpServer *http.Server
	calendar   *calendarapp.Service
}

type Dependencies struct {
	Now            func() time.Time
	ICalClient     *http.Client
	OIDCClient     *http.Client
	RailClient     *http.Client
	ProviderClient *http.Client
	RecipeClient   *http.Client
	ReminderSender notifications.ReminderSender
	MessageSender  notifications.MessageSender
}

type Worker struct {
	Name     string
	Interval time.Duration
	Timeout  time.Duration
	Run      func(context.Context) error
}

func Compose(ctx context.Context, database *sql.DB, cfg config.Config, dependencies Dependencies) (*Server, []Worker) {
	now := dependencies.Now
	if now == nil {
		now = time.Now
	}
	messageSender := dependencies.MessageSender
	icalClient := dependencies.ICalClient
	if icalClient == nil {
		icalClient = services.NewSafeHTTPClient(10 * time.Second)
	}
	recipeClient := dependencies.RecipeClient
	if recipeClient == nil {
		recipeClient = services.NewSafeHTTPClient(15 * time.Second)
	}
	oidcClient := dependencies.OIDCClient
	if oidcClient == nil {
		oidcClient = &http.Client{Timeout: 10 * time.Second}
	}
	railClient := dependencies.RailClient
	if railClient == nil {
		railClient = &http.Client{Timeout: 10 * time.Second}
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
	notificationRepo := repository.NewNotificationRepository(database)
	reminderConfig := services.NewReminderConfig(settingsRepo)
	mealService := mealapp.New(mealPlanRepo)
	householdService := householdapp.New(settingsRepo)
	inventoryService := inventoryapp.New(inventoryRepo)
	stockService := inventoryapp.NewStockService(inventoryRepo)

	choreService := choreapp.NewChoreService(choreRepo, assignmentRepo, userRepo, seriesRepo, choreapp.WithChoreTimezone(services.ChoreTimezone(settingsRepo)), choreapp.WithChoreClock(now))
	icalFetcher := calendarapp.New(ctx, icalSubRepo, services.NewICalAdapter(icalClient), now)
	recipeAdapter := services.NewRecipeExtractor(recipeClient)
	recipeExtractor := recipeapp.New(recipeAdapter, recipeAdapter)
	commuteService := commuteapp.New(services.NewHuxleyClient(cfg.HuxleyBaseURL, cfg.HuxleyAccessToken, railClient, now), now)
	commuteHandler := handlers.NewCommuteHandler(commuteService)
	pairingRepo := repository.NewDisplayPairingRepository(database)
	pairing := displayapp.NewPairing(pairingRepo, services.NewCredentialSecret, repository.HashToken, uuid.NewString, now)
	displaySettings := displayapp.NewSettings(pairingRepo)
	displayPairingHandler := handlers.NewDisplayPairingHandler(displaySettings, pairing)
	displayCommuteHandler := handlers.NewDisplayCommuteHandler(commuteapp.NewDisplayCommute(services.NewDisplayCommutePreferences(displaySettings), services.NewDisplayCommuteTimezone(settingsRepo), services.NewDisplayCommuteRail(commuteService), now))
	scratchpadHandler := handlers.NewScratchpadHandler(scratchapp.New(repository.NewScratchpadRepository(database)))
	identityService := identityapp.New(services.NewOIDCProvider(cfg.OIDCUserInfoURL, oidcClient), userRepo, tokenRepo, services.NewCredentialSecret, repository.HashToken)
	recipeCatalogue := recipeapp.NewCatalogue(recipeRepo, recipeRepo, mealService)
	binService := binsapp.New(services.NewBinScheduleAdapter(settingsRepo), services.BinCalendarParser{}, cfg.PushConfigured() || messageSender != nil, now)
	dashboard := dashboardapp.New(services.NewDashboardLocation(settingsRepo), services.NewDashboardBins(binService), services.NewDashboardChores(choreService), services.NewDashboardMeals(mealService), now)
	apiHandler := handlers.NewAPIHandler(userapp.New(userRepo), categoryapp.New(categoryRepo), services.NewCalendarView(settingsRepo, choreService, icalFetcher, mealService, now), identityapp.NewTokenAdministration(tokenRepo, services.NewCredentialSecret, repository.HashToken), householdService, choreService, mealService, recipeCatalogue, inventoryService, stockService, recipeExtractor, identityService, dashboard, cfg.OIDCClientID, cfg.OIDCIssuer)
	calendarSubscriptionsHandler := handlers.NewCalendarSubscriptionsAPIHandler(calendarapp.NewSubscriptions(icalSubRepo, icalFetcher, nil))
	imageHandler := handlers.NewImageHandler(recipeRepo)
	profileHandler := handlers.NewProfileHandler(userRepo)
	providerClient := dependencies.ProviderClient
	if providerClient == nil {
		providerClient = services.NewSafeHTTPClient(10 * time.Second)
	}
	weatherProvider := weatherapp.New(services.NewOpenMeteo(providerClient, "https://api.open-meteo.com/v1/forecast", now), now)
	drivingProvider := services.NewGoogleDriving(providerClient, cfg.GoogleMapsAPIKey, "https://maps.googleapis.com/maps/api/geocode/json", "https://routes.googleapis.com/directions/v2:computeRoutes")
	departureService := commuteapp.NewDepartureService(icalFetcher, drivingProvider, now)
	householdHandler := handlers.NewHouseholdHandler(householdService, weatherProvider, departureService, shoppingapp.New(repository.NewShoppingRepository(database, repository.NewShoppingInventory), uuid.NewString, now))
	reminderHandler := handlers.NewChoreReminderHandler(notifications.NewReminderManagement(reminderConfig, settingsRepo, notificationRepo, repository.HashToken), cfg.PushConfigured() || messageSender != nil)
	binCollectionsHandler := handlers.NewBinCollectionsHandler(binService)
	broadcastHandler := handlers.NewBroadcastHandler(notifications.NewBroadcastService(repository.NewBroadcastRepository(database), notificationRepo, messageSender), messageSender != nil)
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
	if cfg.APIDocsEnabled {
		router.Get("/api/openapi.json", apidocs.SpecificationHandler)
		router.Get("/api/docs", apidocs.ExplorerHandler)
		router.Get("/api/docs/swagger-ui-bundle.js", apidocs.JavaScriptHandler)
		router.Get("/api/docs/swagger-ui.css", apidocs.StylesheetHandler)
	}
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
		r.Get("/api/display/dashboard", apiHandler.DashboardStats)
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

	reminders := notifications.NewChoreReminderService(reminderConfig, choreRepo, notificationRepo, dependencies.ReminderSender, userRepo)
	bins := notifications.NewBinReminderService(reminderConfig, notificationRepo, messageSender)
	workers := []Worker{
		{Name: "sending chore reminders", Interval: 30 * time.Second, Timeout: 25 * time.Second, Run: func(ctx context.Context) error { return reminders.Check(ctx, now()) }},
		{Name: "sending bin reminders", Interval: 30 * time.Second, Timeout: 25 * time.Second, Run: func(ctx context.Context) error { return bins.Check(ctx, now()) }},
		{Name: "updating overdue chores", Interval: 5 * time.Minute, Run: choreService.UpdateOverdueChores},
	}
	return &Server{router: router, httpServer: &http.Server{Addr: ":" + cfg.Port, Handler: router}, calendar: icalFetcher}, workers
}

func (server *Server) Start() error {
	slog.Info("starting server", "address", server.httpServer.Addr)
	return server.httpServer.ListenAndServe()
}
func (server *Server) Shutdown(ctx context.Context) error {
	httpError := server.httpServer.Shutdown(ctx)
	calendarError := server.calendar.Shutdown(ctx)
	return errors.Join(httpError, calendarError)
}
func (server *Server) Close() error { return server.httpServer.Close() }

func (server *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	server.router.ServeHTTP(w, r)
}
