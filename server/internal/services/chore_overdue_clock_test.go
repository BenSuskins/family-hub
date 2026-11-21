package services_test

import (
	"context"
	choreapp "github.com/bensuskins/family-hub/internal/application/chores"
	"github.com/bensuskins/family-hub/internal/models"
	"testing"
	"time"
)

func TestOverdueWorkerUsesInjectedClock(t *testing.T) {
	_, chores, assignments, users, series := setupChoreServiceWithSeries(t)
	user := createUsers(t, users, 1)[0]
	now := time.Date(2001, time.June, 15, 12, 0, 0, 0, time.UTC)
	past := now.AddDate(0, 0, -1)
	future := now.AddDate(0, 0, 1)
	for _, entry := range []struct {
		name string
		due  *time.Time
	}{{"past", &past}, {"future", &future}} {
		if _, err := chores.Create(context.Background(), models.Chore{Name: entry.name, CreatedByUserID: user.ID, DueDate: entry.due, Status: models.ChoreStatusPending}); err != nil {
			t.Fatal(err)
		}
	}
	service := choreapp.NewChoreService(chores, assignments, users, series, choreapp.WithChoreClock(func() time.Time { return now }))
	if err := service.UpdateOverdueChores(context.Background()); err != nil {
		t.Fatal(err)
	}
	items, err := chores.FindAll(context.Background(), choreapp.ChoreFilter{})
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range items {
		want := models.ChoreStatusPending
		if item.Name == "past" {
			want = models.ChoreStatusOverdue
		}
		if item.Status != want {
			t.Fatalf("%s status = %s, want %s", item.Name, item.Status, want)
		}
	}
}
