package handlers

import (
	"time"

	binsapp "github.com/bensuskins/family-hub/internal/application/bins"
	dashboardapp "github.com/bensuskins/family-hub/internal/application/dashboard"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/services"
)

func testDashboard(handler *APIHandler, settings repository.SettingsRepository) *dashboardapp.Service {
	var location dashboardapp.LocationSource
	var bins dashboardapp.BinSource
	if settings != nil {
		location = services.NewDashboardLocation(settings)
		bins = services.NewDashboardBins(binsapp.New(services.NewBinScheduleAdapter(settings), services.BinCalendarParser{}, false, time.Now))
	}
	var meals dashboardapp.MealSource
	if handler.mealService != nil {
		meals = services.NewDashboardMeals(handler.mealService)
	}
	return dashboardapp.New(location, bins, services.NewDashboardChores(handler.choreService), meals, time.Now)
}
