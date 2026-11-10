package commute_test

import (
	"context"
	"testing"
	"time"

	"github.com/bensuskins/family-hub/internal/application/commute"
	"github.com/bensuskins/family-hub/internal/models"
)

type trainProvider struct {
	calls               int
	origin, destination string
}

func (provider *trainProvider) FetchTrains(_ context.Context, origin, destination string, count int) (models.CommuteBoard, error) {
	provider.calls++
	provider.origin = origin
	provider.destination = destination
	return models.CommuteBoard{Trains: []models.CommuteTrain{{ServiceID: "first"}}}, nil
}
func TestFetchTrainsCachesNormalizedRouteForThirtySeconds(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	provider := &trainProvider{}
	service := commute.New(provider, func() time.Time { return now })
	first, err := service.FetchTrains(context.Background(), " kel ", "lst", 5)
	if err != nil || provider.origin != "KEL" || provider.destination != "LST" {
		t.Fatalf("fetch: %v %+v", err, provider)
	}
	first.Trains[0].ServiceID = "changed"
	cached, err := service.FetchTrains(context.Background(), "KEL", "LST", 5)
	if err != nil || cached.Trains[0].ServiceID != "first" || provider.calls != 1 {
		t.Fatalf("cache: %+v, calls %d, error %v", cached, provider.calls, err)
	}
	now = now.Add(30 * time.Second)
	_, err = service.FetchTrains(context.Background(), "KEL", "LST", 5)
	if err != nil || provider.calls != 2 {
		t.Fatalf("expiry: calls %d, error %v", provider.calls, err)
	}
}
