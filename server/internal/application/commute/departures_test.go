package commute_test

import (
	"context"
	"testing"
	"time"

	"github.com/bensuskins/family-hub/internal/application/commute"
	"github.com/bensuskins/family-hub/internal/models"
)

type calendarFake struct{ events []models.Event }

func (calendar calendarFake) FetchForRange(context.Context, time.Time, time.Time) ([]models.Event, error) {
	return calendar.events, nil
}

type drivingFake struct{ calls int }

func (driving *drivingFake) DrivingSeconds(_ context.Context, _ models.HomeInfo, _ string) (int, error) {
	driving.calls++
	return 600, nil
}
func TestDeparturesSkipsOnlineEventsAndAppliesArrivalMargin(t *testing.T) {
	now := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	calendar := calendarFake{events: []models.Event{{ID: "office", Location: "Office", StartTime: now.Add(2 * time.Hour)}, {ID: "online", Location: "https://example.com", StartTime: now.Add(time.Hour)}}}
	driving := &drivingFake{}
	service := commute.NewDepartureService(calendar, driving, func() time.Time { return now })
	result, err := service.Departures(context.Background(), models.HomeInfo{Timezone: "UTC", ArrivalMarginMinutes: 5})
	if err != nil || driving.calls != 1 || len(result.Items) != 1 || result.Items[0].EventID != "office" || !result.Items[0].LeaveAt.Equal(now.Add(105*time.Minute)) {
		t.Fatalf("departures: %+v, calls %d, error %v", result, driving.calls, err)
	}
	if calendar.events[0].ID != "office" {
		t.Fatal("operation mutated the calendar events")
	}
}
