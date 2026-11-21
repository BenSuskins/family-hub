package server_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bensuskins/family-hub/internal/config"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/server"
	"github.com/bensuskins/family-hub/internal/testutil"
)

func TestCompositionSharesClockAcrossHTTPAndWorkers(t *testing.T) {
	ctx := context.Background()
	database := testutil.NewTestDatabase(t)
	user, err := repository.NewUserRepository(database).Create(ctx, models.User{OIDCSubject: "composition-clock", Email: "clock@example.com", Name: "Clock", Role: models.RoleMember})
	if err != nil {
		t.Fatal(err)
	}
	_, err = repository.NewAPITokenRepository(database).Create(ctx, models.APIToken{Name: "test", TokenHash: repository.HashToken("composition-token"), Scope: models.TokenScopeAPI, CreatedByUserID: user.ID})
	if err != nil {
		t.Fatal(err)
	}
	due := time.Date(2001, time.March, 15, 12, 0, 0, 0, time.UTC)
	chore, err := repository.NewChoreRepository(database).Create(ctx, models.Chore{Name: "Future chore", CreatedByUserID: user.ID, DueDate: &due})
	if err != nil {
		t.Fatal(err)
	}
	fixed := time.Date(2001, time.March, 5, 12, 0, 0, 0, time.UTC)
	application, workers := server.Compose(ctx, database, config.Config{}, server.Dependencies{Now: func() time.Time { return fixed }})
	t.Cleanup(func() { _ = application.Shutdown(context.Background()) })
	request := httptest.NewRequest(http.MethodGet, "/api/calendar", nil)
	request.Header.Set("Authorization", "Bearer composition-token")
	response := httptest.NewRecorder()
	application.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("calendar response = %d: %s", response.Code, response.Body.String())
	}
	var result struct {
		Chores []struct{ ID string } `json:"chores"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Chores) != 1 || result.Chores[0].ID != chore.ID {
		t.Fatalf("calendar does not use injected month: %s", response.Body.String())
	}
	wantWorkers := map[string]struct{ interval, timeout time.Duration }{
		"sending chore reminders": {30 * time.Second, 25 * time.Second},
		"sending bin reminders":   {30 * time.Second, 25 * time.Second},
		"updating overdue chores": {5 * time.Minute, 0},
	}
	if len(workers) != len(wantWorkers) {
		t.Fatalf("worker count = %d, want %d", len(workers), len(wantWorkers))
	}
	var overdue *server.Worker
	for index := range workers {
		worker := &workers[index]
		want, found := wantWorkers[worker.Name]
		if !found || worker.Interval != want.interval || worker.Timeout != want.timeout {
			t.Fatalf("unexpected worker: %+v", worker)
		}
		delete(wantWorkers, worker.Name)
		if worker.Name == "updating overdue chores" {
			overdue = worker
		}
	}
	if overdue == nil {
		t.Fatal("overdue worker missing")
	}
	if err := overdue.Run(ctx); err != nil {
		t.Fatal(err)
	}
	saved, err := repository.NewChoreRepository(database).FindByID(ctx, chore.ID)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Status != models.ChoreStatusPending {
		t.Fatalf("worker did not use injected clock: status %q", saved.Status)
	}
}

type cancelableCalendarTransport struct {
	started   chan struct{}
	stopped   chan struct{}
	calls     atomic.Int32
	startOnce sync.Once
	stopOnce  sync.Once
}

func (transport *cancelableCalendarTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	transport.calls.Add(1)
	transport.startOnce.Do(func() { close(transport.started) })
	<-request.Context().Done()
	transport.stopOnce.Do(func() { close(transport.stopped) })
	return nil, request.Context().Err()
}

func TestComposedShutdownCancelsWorkerAndCalendarRefresh(t *testing.T) {
	root, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	database := testutil.NewTestDatabase(t)
	ctx := context.Background()
	user, err := repository.NewUserRepository(database).Create(ctx, models.User{OIDCSubject: "composition-shutdown", Email: "shutdown@example.com", Name: "Shutdown", Role: models.RoleMember})
	if err != nil {
		t.Fatal(err)
	}
	_, err = repository.NewAPITokenRepository(database).Create(ctx, models.APIToken{Name: "test", TokenHash: repository.HashToken("shutdown-token"), Scope: models.TokenScopeAPI, CreatedByUserID: user.ID})
	if err != nil {
		t.Fatal(err)
	}
	subscriptions := repository.NewICalSubscriptionRepository(database)
	if err := subscriptions.Create(ctx, models.ICalSubscription{ID: "calendar", Name: "Calendar", URL: "https://8.8.8.8/calendar.ics"}); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2001, time.March, 5, 12, 0, 0, 0, time.UTC)
	calendar := "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nEND:VCALENDAR\r\n"
	if err := subscriptions.UpdateCache(ctx, "calendar", calendar, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	transport := &cancelableCalendarTransport{started: make(chan struct{}), stopped: make(chan struct{})}
	application, workers := server.Compose(root, database, config.Config{}, server.Dependencies{Now: func() time.Time { return now }, ICalClient: &http.Client{Transport: transport}})
	request := httptest.NewRequest(http.MethodGet, "/api/calendar", nil)
	request.Header.Set("Authorization", "Bearer shutdown-token")
	response := httptest.NewRecorder()
	application.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("calendar response = %d: %s", response.Code, response.Body.String())
	}
	select {
	case <-transport.started:
	case <-time.After(5 * time.Second):
		t.Fatal("calendar refresh did not start")
	}
	transaction, err := database.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = transaction.Rollback() })
	var reminder *server.Worker
	for index := range workers {
		if workers[index].Name == "sending chore reminders" {
			reminder = &workers[index]
		}
	}
	if reminder == nil {
		t.Fatal("chore reminder worker missing")
	}
	started := make(chan struct{})
	done := make(chan error, 1)
	go func() { close(started); done <- reminder.Run(root) }()
	<-started
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("worker error = %v, want context canceled", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("worker did not stop after cancellation")
	}
	if err := transaction.Rollback(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-transport.stopped:
	case <-time.After(5 * time.Second):
		t.Fatal("calendar refresh did not stop after cancellation")
	}
	shutdown, stop := context.WithTimeout(context.Background(), 5*time.Second)
	defer stop()
	if err := application.Shutdown(shutdown); err != nil {
		t.Fatal(err)
	}
	response = httptest.NewRecorder()
	application.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("calendar after shutdown = %d: %s", response.Code, response.Body.String())
	}
	if calls := transport.calls.Load(); calls != 1 {
		t.Fatalf("calendar refresh calls after shutdown = %d", calls)
	}
}
