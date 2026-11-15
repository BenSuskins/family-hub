package handlers

import (
	binsapp "github.com/bensuskins/family-hub/internal/application/bins"
	dashboardapp "github.com/bensuskins/family-hub/internal/application/dashboard"
	"github.com/bensuskins/family-hub/internal/services"
	"time"
)

func testDashboard(handler *APIHandler) *dashboardapp.Service {
	var location dashboardapp.LocationSource
	var bins dashboardapp.BinSource
	if handler.settingsRepo != nil {
		location = services.NewDashboardLocation(handler.settingsRepo)
		bins = services.NewDashboardBins(binsapp.New(services.NewBinScheduleAdapter(handler.settingsRepo), services.BinCalendarParser{}, false, time.Now))
	}
	var meals dashboardapp.MealSource
	if handler.mealService != nil {
		meals = services.NewDashboardMeals(handler.mealService)
	}
	return dashboardapp.New(location, bins, services.NewDashboardChores(handler.choreService), meals, time.Now)
}
