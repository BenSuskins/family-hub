package services_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/services"
	"github.com/bensuskins/family-hub/internal/testutil"
)

func TestCurrentTaskResolvesOnceAndRotatesFromOwner(t *testing.T) {
	connection := testutil.NewTestDatabase(t)
	chores := repository.NewChoreRepository(connection)
	assignments := repository.NewChoreAssignmentRepository(connection)
	people := repository.NewUserRepository(connection)
	definitions := repository.NewChoreSeriesRepository(connection)
	now := time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	service := services.NewChoreService(chores, assignments, people, definitions, services.WithChoreClock(func() time.Time { return now }))
	users := createUsers(t, people, 2)
	due := time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC)
	task := newRecurringChore(t, chores, definitions, models.ChoreSeries{Name: "Bins", RecurrenceType: models.RecurrenceDaily}, models.Chore{Name: "Bins", CreatedByUserID: users[0].ID, AssignedToUserID: &users[0].ID, DueDate: &due})
	if _, err := assignments.Create(context.Background(), models.ChoreAssignment{ChoreID: task.ID, UserID: users[0].ID}); err != nil {
		t.Fatal(err)
	}
	var workers sync.WaitGroup
	results := make(chan error, 2)
	workers.Add(2)
	go func() {
		defer workers.Done()
		results <- service.CompleteChore(context.Background(), task.ID, users[1].ID)
	}()
	go func() { defer workers.Done(); results <- service.SkipChore(context.Background(), task.ID, users[1].ID) }()
	workers.Wait()
	close(results)
	successes := 0
	for err := range results {
		if err == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("successful actions: %d", successes)
	}
	current, err := chores.FindAll(context.Background(), repository.ChoreFilter{Statuses: []models.ChoreStatus{models.ChoreStatusPending, models.ChoreStatusOverdue}})
	if err != nil {
		t.Fatal(err)
	}
	if len(current) != 1 {
		t.Fatalf("current tasks: %d", len(current))
	}
	if current[0].DueDate.Format("2006-01-02") != "2026-10-19" {
		t.Fatal(current[0].DueDate)
	}
	if current[0].AssignedToUserID == nil || *current[0].AssignedToUserID != users[1].ID {
		t.Fatal("rotation did not advance from owner")
	}
	if _, err := service.Forecast(context.Background(), now, now.AddDate(0, 0, 7)); err != nil {
		t.Fatal(err)
	}
	currentAfter, _ := chores.FindAll(context.Background(), repository.ChoreFilter{Statuses: []models.ChoreStatus{models.ChoreStatusPending}})
	if len(currentAfter) != 1 {
		t.Fatal("forecast created tasks")
	}
}

func TestTaskDatesPreserveRepeatRules(t *testing.T) {
	cases := []struct {
		name, now, due, rule, config, want string
		afterCompletion                    bool
	}{
		{name: "early daily", now: "2026-10-18T12:00:00Z", due: "2026-10-20", rule: "daily", want: "2026-10-21"},
		{name: "late interval preserves cadence", now: "2026-10-18T12:00:00Z", due: "2026-10-10", rule: "daily", config: `{"interval":3}`, want: "2026-10-19"},
		{name: "month end", now: "2026-01-31T12:00:00Z", due: "2026-01-31", rule: "monthly", config: `{"day_of_month":31}`, want: "2026-02-28"},
		{name: "multiple weekdays", now: "2026-01-05T12:00:00Z", due: "2026-01-05", rule: "weekly", config: `{"interval":2,"days":["monday","thursday"]}`, want: "2026-01-08"},
		{name: "late fortnight", now: "2026-01-14T12:00:00Z", due: "2026-01-05", rule: "weekly", config: `{"interval":2,"days":["monday","thursday"]}`, want: "2026-01-19"},
		{name: "after completion", now: "2026-10-18T12:00:00Z", due: "2026-10-10", rule: "daily", config: `{"interval":3}`, want: "2026-10-21", afterCompletion: true},
		{name: "daylight saving", now: "2026-03-29T12:00:00Z", due: "2026-03-28", rule: "daily", want: "2026-03-30"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			connection := testutil.NewTestDatabase(t)
			chores := repository.NewChoreRepository(connection)
			assignments := repository.NewChoreAssignmentRepository(connection)
			people := repository.NewUserRepository(connection)
			definitions := repository.NewChoreSeriesRepository(connection)
			now, _ := time.Parse(time.RFC3339, test.now)
			due, _ := time.Parse("2006-01-02", test.due)
			service := services.NewChoreService(chores, assignments, people, definitions, services.WithChoreClock(func() time.Time { return now }))
			users := createUsers(t, people, 1)
			task := newRecurringChore(t, chores, definitions, models.ChoreSeries{Name: "Routine", RecurrenceType: models.RecurrenceType(test.rule), RecurrenceValue: test.config, RecurOnComplete: test.afterCompletion}, models.Chore{Name: "Routine", CreatedByUserID: users[0].ID, AssignedToUserID: &users[0].ID, DueDate: &due})
			if err := service.CompleteChore(context.Background(), task.ID, users[0].ID); err != nil {
				t.Fatal(err)
			}
			current, err := chores.FindAll(context.Background(), repository.ChoreFilter{Statuses: []models.ChoreStatus{models.ChoreStatusPending}})
			if err != nil {
				t.Fatal(err)
			}
			if len(current) != 1 || current[0].DueDate.Format("2006-01-02") != test.want {
				t.Fatalf("expected %s, got %+v", test.want, current)
			}
		})
	}
}

func TestResolutionFailureRollsBackTheTask(t *testing.T) {
	service, chores, assignments, people, definitions := setupChoreServiceWithSeries(t)
	users := createUsers(t, people, 1)
	due := time.Now()
	task := newRecurringChore(t, chores, definitions, models.ChoreSeries{Name: "Invalid", RecurrenceType: models.RecurrenceDaily, RecurrenceValue: "invalid"}, models.Chore{Name: "Invalid", CreatedByUserID: users[0].ID, AssignedToUserID: &users[0].ID, DueDate: &due})
	if _, err := assignments.Create(context.Background(), models.ChoreAssignment{ChoreID: task.ID, UserID: users[0].ID}); err != nil {
		t.Fatal(err)
	}
	if err := service.CompleteChore(context.Background(), task.ID, users[0].ID); err == nil {
		t.Fatal("invalid rule accepted")
	}
	current, err := chores.FindByID(context.Background(), task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.Status != models.ChoreStatusPending || current.CompletedAt != nil {
		t.Fatal("failed completion changed task")
	}
	history, err := assignments.FindByChoreID(context.Background(), task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 1 || history[0].Status != models.AssignmentStatusAssigned {
		t.Fatalf("failed completion changed assignment: %+v", history)
	}
}

func TestForecastHonoursLimitsWithoutCreatingTasks(t *testing.T) {
	connection := testutil.NewTestDatabase(t)
	chores := repository.NewChoreRepository(connection)
	people := repository.NewUserRepository(connection)
	definitions := repository.NewChoreSeriesRepository(connection)
	now := time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	service := services.NewChoreService(chores, repository.NewChoreAssignmentRepository(connection), people, definitions, services.WithChoreClock(func() time.Time { return now }))
	users := createUsers(t, people, 2)
	due := time.Date(2026, 10, 18, 0, 0, 0, 0, time.UTC)
	limit := 3
	task := newRecurringChore(t, chores, definitions, models.ChoreSeries{Name: "Bins", RecurrenceType: models.RecurrenceDaily, RecurrenceCount: &limit}, models.Chore{Name: "Bins", CreatedByUserID: users[0].ID, AssignedToUserID: &users[0].ID, DueDate: &due})
	predictions, err := service.Forecast(context.Background(), now, now.AddDate(0, 0, 7))
	if err != nil {
		t.Fatal(err)
	}
	if len(predictions) != 2 || predictions[0].DueDate.Format("2006-01-02") != "2026-10-19" || !predictions[0].IsForecast {
		t.Fatalf("forecast: %+v", predictions)
	}
	if *predictions[0].AssignedToUserID != users[1].ID {
		t.Fatal("forecast does not rotate from current owner")
	}
	if err := service.CompleteChore(context.Background(), predictions[0].ID, users[0].ID); err == nil {
		t.Fatal("forecast can complete")
	}
	current, err := chores.FindAll(context.Background(), repository.ChoreFilter{SeriesID: task.SeriesID})
	if err != nil {
		t.Fatal(err)
	}
	if len(current) != 1 {
		t.Fatal("forecast created persisted tasks")
	}
	definition, err := definitions.FindByID(context.Background(), *task.SeriesID)
	if err != nil {
		t.Fatal(err)
	}
	definition.RecurOnComplete = true
	if err := definitions.Update(context.Background(), *definition); err != nil {
		t.Fatal(err)
	}
	predictions, err = service.Forecast(context.Background(), now, now.AddDate(0, 0, 7))
	if err != nil {
		t.Fatal(err)
	}
	if len(predictions) != 0 {
		t.Fatal("completion-based dates must remain unknown")
	}
}

func TestUndatedTasksAppearOnHomeAndCanRepeat(t *testing.T) {
	service, chores, _, people, _ := setupChoreServiceWithSeries(t)
	users := createUsers(t, people, 1)
	task, err := service.CreateChore(context.Background(), models.Chore{Name: "Undated", CreatedByUserID: users[0].ID, RecurrenceType: models.RecurrenceDaily}, nil)
	if err != nil {
		t.Fatal(err)
	}
	today, overdue, err := service.DueTasks(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(today) != 1 || len(overdue) != 0 {
		t.Fatalf("undated task missing from Home: %+v %+v", today, overdue)
	}
	if err := service.SkipChore(context.Background(), task.ID, users[0].ID); err != nil {
		t.Fatal(err)
	}
	pending, err := chores.FindAll(context.Background(), repository.ChoreFilter{Statuses: []models.ChoreStatus{models.ChoreStatusPending}})
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 1 || pending[0].DueDate == nil {
		t.Fatal("undated repeating task has no successor")
	}
}
