package services_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/bensuskins/family-hub/internal/services"
)

func TestHuxleyLive(t *testing.T) {
	if os.Getenv("COMMUTE_LIVE_TEST") != "true" {
		t.Skip("set COMMUTE_LIVE_TEST=true to check the live provider")
	}
	token := os.Getenv("HUXLEY_ACCESS_TOKEN")
	if token == "" {
		t.Fatal("HUXLEY_ACCESS_TOKEN is required for the live check")
	}
	baseURL := os.Getenv("HUXLEY_BASE_URL")
	if baseURL == "" {
		baseURL = "https://huxley2.azurewebsites.net"
	}
	client := services.NewHuxleyClient(baseURL, token, nil)
	for _, route := range []struct{ origin, destination string }{{"KEL", "LST"}, {"LST", "KEL"}} {
		t.Run(route.origin+"-"+route.destination, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			board, err := client.FetchTrains(ctx, route.origin, route.destination, 5)
			if err != nil {
				t.Fatal(err)
			}
			if board.GeneratedAt.IsZero() || board.FetchedAt.IsZero() || board.Trains == nil {
				t.Fatal("provider returned an incomplete board")
			}
			for _, train := range board.Trains {
				if train.ServiceID == "" || train.ScheduledDeparture == nil {
					t.Fatal("provider returned a train without its ID or departure time")
				}
				if train.Status != "cancelled" && train.ScheduledArrival == nil {
					t.Fatal("provider returned a train without its destination calling time")
				}
			}
			t.Logf("%s to %s: %d trains; provider timestamp %s", board.Origin.Code, board.Destination.Code, len(board.Trains), board.GeneratedAt.Format(time.RFC3339))
		})
	}
}
