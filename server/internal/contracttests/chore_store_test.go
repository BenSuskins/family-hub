package contracttests

import (
	"context"
	"errors"
	"testing"
	"time"

	choreapp "github.com/bensuskins/family-hub/internal/application/chores"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
	"github.com/bensuskins/family-hub/internal/testutil/chorefakes"
)

type choreFixture struct {
	store       choreapp.ChoreRepository
	user        models.User
	assignments interface {
		FindByChoreID(context.Context, string) ([]models.ChoreAssignment, error)
	}
}

func TestChoreStoreContract(t *testing.T) {
	factories := map[string]func(*testing.T) choreFixture{
		"sqlite": func(t *testing.T) choreFixture {
			db := testutil.NewTestDatabase(t)
			user, err := repository.NewUserRepository(db).Create(context.Background(), models.User{OIDCSubject: "contract", Email: "contract@example.com", Name: "Member", Role: models.RoleMember})
			if err != nil {
				t.Fatal(err)
			}
			return choreFixture{store: repository.NewChoreRepository(db), user: user, assignments: repository.NewChoreAssignmentRepository(db)}
		},
		"fake": func(t *testing.T) choreFixture {
			user := models.User{ID: "user-contract", OIDCSubject: "contract", Email: "contract@example.com", Name: "Member", Role: models.RoleMember}
			store := chorefakes.New([]models.User{user})
			return choreFixture{store: store, user: user, assignments: store.Assignments()}
		},
	}
	for name, factory := range factories {
		t.Run(name, func(t *testing.T) {
			fixture := factory(t)
			ctx := context.Background()
			user := fixture.user

			if _, err := fixture.store.FindByID(ctx, "missing-chore"); !errors.Is(err, choreapp.ErrChoreNotFound) {
				t.Fatalf("missing chore: %v", err)
			}
			due := time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)
			task, err := fixture.store.Create(ctx, models.Chore{Name: "Wash up", CreatedByUserID: user.ID, Status: models.ChoreStatusPending, DueDate: &due})
			if err != nil {
				t.Fatal(err)
			}
			original := due
			due = due.AddDate(0, 0, 1)
			saved, err := fixture.store.FindByID(ctx, task.ID)
			if err != nil {
				t.Fatal(err)
			}
			if saved.DueDate == nil || !saved.DueDate.Equal(original) {
				t.Fatalf("stored date changed: %+v", saved.DueDate)
			}
			*saved.DueDate = due
			saved, err = fixture.store.FindByID(ctx, task.ID)
			if err != nil || !saved.DueDate.Equal(original) {
				t.Fatalf("snapshot changed stored value: %+v %v", saved.DueDate, err)
			}

			changedOwner := saved
			changedOwner.CreatedByUserID = "other-user"
			if err := fixture.store.Update(ctx, changedOwner); err != nil {
				t.Fatal(err)
			}
			saved, err = fixture.store.FindByID(ctx, task.ID)
			if err != nil || saved.CreatedByUserID != user.ID {
				t.Fatalf("owner changed on update: %+v %v", saved, err)
			}

			resolvedAt := fixedChoreTime()
			resolvedUser := user.ID
			legacy, err := fixture.store.Create(ctx, models.Chore{Name: "Imported", CreatedByUserID: user.ID, ResolvedAt: &resolvedAt, ResolvedByUserID: &resolvedUser})
			if err != nil {
				t.Fatal(err)
			}
			legacySaved, err := fixture.store.FindByID(ctx, legacy.ID)
			if err != nil || legacySaved.ResolvedAt != nil || legacySaved.ResolvedByUserID != nil {
				t.Fatalf("create resolution fields: %+v %v", legacySaved, err)
			}
			abort := errors.New("rollback")
			err = fixture.store.Transact(ctx, func(chores choreapp.ChoreRepository, assignments choreapp.ChoreAssignmentRepository, users choreapp.UserRepository, definitions choreapp.ChoreSeriesRepository) error {
				saved.Name = "Changed"
				if err := chores.Update(ctx, saved); err != nil {
					return err
				}
				if _, err := assignments.Create(ctx, models.ChoreAssignment{ChoreID: task.ID, UserID: user.ID, Status: models.AssignmentStatusAssigned}); err != nil {
					return err
				}
				if _, err := definitions.Create(ctx, models.ChoreSeries{ID: task.ID, Name: "Series", CreatedByUserID: user.ID}); err != nil {
					return err
				}
				return abort
			})
			if !errors.Is(err, abort) {
				t.Fatalf("rollback error %v", err)
			}
			saved, err = fixture.store.FindByID(ctx, task.ID)
			if err != nil || saved.Name != "Wash up" {
				t.Fatalf("rollback: %+v %v", saved, err)
			}

			assignments, err := fixture.assignments.FindByChoreID(ctx, task.ID)
			if err != nil || len(assignments) != 0 {
				t.Fatalf("assignment rollback: %+v %v", assignments, err)
			}
			var expired choreapp.ChoreRepository
			err = fixture.store.Transact(ctx, func(chores choreapp.ChoreRepository, assignments choreapp.ChoreAssignmentRepository, users choreapp.UserRepository, definitions choreapp.ChoreSeriesRepository) error {
				expired = chores
				series, err := definitions.FindByID(ctx, task.ID)
				if err != nil {
					return err
				}
				if series != nil {
					t.Fatal("rolled-back series remained")
				}
				saved.Name = "Clean kitchen"
				return chores.Update(ctx, saved)
			})
			if err != nil {
				t.Fatal(err)
			}
			saved, err = fixture.store.FindByID(ctx, task.ID)
			if err != nil || saved.Name != "Clean kitchen" {
				t.Fatalf("commit: %+v %v", saved, err)
			}

			saved.Name = "After commit"
			if err := expired.Update(ctx, saved); err == nil {
				t.Fatal("transaction remained writable after commit")
			}
			saved, err = fixture.store.FindByID(ctx, task.ID)
			if err != nil || saved.Name != "Clean kitchen" {
				t.Fatalf("expired transaction changed state: %+v %v", saved, err)
			}

			if _, err := fixture.store.Create(ctx, models.Chore{Name: "Invalid", CreatedByUserID: "missing-user"}); err == nil {
				t.Fatal("accepted a missing creator")
			}

			if _, err := fixture.store.Create(ctx, models.Chore{Name: "Invalid status", CreatedByUserID: user.ID, Status: models.ChoreStatus("invalid")}); err == nil {
				t.Fatal("accepted invalid chore status")
			}
			missingCategory := "missing-category"
			if _, err := fixture.store.Create(ctx, models.Chore{Name: "Invalid category", CreatedByUserID: user.ID, CategoryID: &missingCategory}); err == nil {
				t.Fatal("accepted missing category")
			}
			if err := fixture.store.Transact(ctx, func(_ choreapp.ChoreRepository, assignments choreapp.ChoreAssignmentRepository, _ choreapp.UserRepository, definitions choreapp.ChoreSeriesRepository) error {
				if _, err := assignments.Create(ctx, models.ChoreAssignment{ChoreID: "missing-chore", UserID: user.ID}); err == nil {
					t.Error("accepted an assignment for a missing chore")
				}
				if _, err := definitions.Create(ctx, models.ChoreSeries{Name: "Invalid", CreatedByUserID: "missing-user"}); err == nil {
					t.Error("accepted a series with a missing creator")
				}

				if _, err := definitions.Create(ctx, models.ChoreSeries{Name: "Invalid recurrence", CreatedByUserID: user.ID, RecurrenceType: models.RecurrenceType("invalid")}); err == nil {
					t.Error("accepted invalid recurrence")
				}
				if _, err := assignments.Create(ctx, models.ChoreAssignment{ChoreID: task.ID, UserID: user.ID, Status: models.AssignmentStatus("invalid")}); err == nil {
					t.Error("accepted invalid assignment status")
				}
				return nil
			}); err != nil {
				t.Fatal(err)
			}

			err = fixture.store.Transact(ctx, func(_ choreapp.ChoreRepository, _ choreapp.ChoreAssignmentRepository, _ choreapp.UserRepository, definitions choreapp.ChoreSeriesRepository) error {
				initial := []string{user.ID}
				created, err := definitions.Create(ctx, models.ChoreSeries{ID: "contract-series", Name: "Series", CreatedByUserID: user.ID, EligibleAssignees: initial})
				if err != nil {
					return err
				}
				savedSeries, err := definitions.FindByID(ctx, created.ID)
				if err != nil {
					return err
				}
				if len(savedSeries.EligibleAssignees) != 0 {
					t.Error("creation persisted eligible users without a separate assignment")
				}
				if err := definitions.SetEligibleAssignees(ctx, created.ID, initial); err != nil {
					return err
				}
				initial[0] = "mutated"
				savedSeries, err = definitions.FindByID(ctx, created.ID)
				if err != nil {
					return err
				}
				if len(savedSeries.EligibleAssignees) != 1 || savedSeries.EligibleAssignees[0] != user.ID {
					t.Error("series eligible users changed through input")
				}
				savedSeries.EligibleAssignees[0] = "mutated"
				savedSeries, err = definitions.FindByID(ctx, created.ID)
				if err != nil {
					return err
				}
				if len(savedSeries.EligibleAssignees) != 1 || savedSeries.EligibleAssignees[0] != user.ID {
					t.Error("series eligible users changed through output")
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			members := []string{user.ID}
			if err := fixture.store.SetEligibleAssignees(ctx, task.ID, members); err != nil {
				t.Fatal(err)
			}

			if err := fixture.store.SetEligibleAssignees(ctx, task.ID, []string{user.ID, user.ID}); err == nil {
				t.Fatal("accepted duplicate eligible users")
			}
			members[0] = "mutated"
			eligible, err := fixture.store.GetEligibleAssignees(ctx, task.ID)
			if err != nil || len(eligible) != 1 || eligible[0] != user.ID {
				t.Fatalf("eligible input alias: %v %v", eligible, err)
			}
			eligible[0] = "mutated"
			eligible, err = fixture.store.GetEligibleAssignees(ctx, task.ID)
			if err != nil || len(eligible) != 1 || eligible[0] != user.ID {
				t.Fatalf("eligible output alias: %v %v", eligible, err)
			}
			earlier := original.Add(-24 * time.Hour)
			second, err := fixture.store.Create(ctx, models.Chore{Name: "Earlier", CreatedByUserID: user.ID, DueDate: &earlier})
			if err != nil {
				t.Fatal(err)
			}
			tasks, err := fixture.store.FindAll(ctx, choreapp.ChoreFilter{OrderBy: choreapp.OrderDueDate, Limit: 1})
			if err != nil || len(tasks) != 1 || tasks[0].ID != second.ID {
				t.Fatalf("ordering and limit: %+v %v", tasks, err)
			}
			missingStatus := models.ChoreStatusSkipped
			tasks, err = fixture.store.FindAll(ctx, choreapp.ChoreFilter{Status: &missingStatus})
			if err != nil || len(tasks) != 0 {
				t.Fatalf("status filter: %+v %v", tasks, err)
			}
			cancelled, cancel := context.WithCancel(ctx)
			cancel()
			called := false
			err = fixture.store.Transact(cancelled, func(choreapp.ChoreRepository, choreapp.ChoreAssignmentRepository, choreapp.UserRepository, choreapp.ChoreSeriesRepository) error {
				called = true
				return nil
			})
			if !errors.Is(err, context.Canceled) || called {
				t.Fatalf("cancelled transaction: %v called=%v", err, called)
			}
		})
	}
}

func TestChoreLifecycleContract(t *testing.T) {
	factories := map[string]func(*testing.T) (*choreapp.ChoreService, choreapp.ChoreRepository, []models.User){
		"sqlite": func(t *testing.T) (*choreapp.ChoreService, choreapp.ChoreRepository, []models.User) {
			database := testutil.NewTestDatabase(t)
			people := repository.NewUserRepository(database)
			users := []models.User{}
			for _, name := range []string{"Alice", "Bob"} {
				user, err := people.Create(context.Background(), models.User{OIDCSubject: name, Email: name + "@example.com", Name: name, Role: models.RoleMember})
				if err != nil {
					t.Fatal(err)
				}
				users = append(users, user)
			}
			store := repository.NewChoreRepository(database)
			return choreapp.NewChoreService(store, repository.NewChoreAssignmentRepository(database), people, repository.NewChoreSeriesRepository(database), choreapp.WithChoreClock(fixedChoreTime)), store, users
		},
		"fake": func(t *testing.T) (*choreapp.ChoreService, choreapp.ChoreRepository, []models.User) {
			users := []models.User{{ID: "alice", Name: "Alice"}, {ID: "bob", Name: "Bob"}}
			store := chorefakes.New(users)
			return choreapp.NewChoreService(store, store.Assignments(), store.Users(), store.Series(), choreapp.WithChoreClock(fixedChoreTime)), store, users
		},
	}
	for name, factory := range factories {
		t.Run(name, func(t *testing.T) {
			service, store, users := factory(t)
			ctx := context.Background()
			due := fixedChoreTime()
			task, err := service.CreateChore(ctx, models.Chore{Name: "Dishes", CreatedByUserID: users[0].ID, DueDate: &due, RecurrenceType: models.RecurrenceDaily, RecurOnComplete: true}, nil)
			if err != nil {
				t.Fatal(err)
			}
			if task.AssignedToUserID == nil {
				t.Fatal("first occurrence not assigned")
			}
			first := *task.AssignedToUserID
			if err := service.CompleteChore(ctx, task.ID, users[0].ID); err != nil {
				t.Fatal(err)
			}
			if !errors.Is(service.CompleteChore(ctx, task.ID, users[0].ID), choreapp.ErrChoreAlreadyComplete) {
				t.Fatal("repeated completion succeeded")
			}
			tasks, err := store.FindAll(ctx, choreapp.ChoreFilter{Statuses: []models.ChoreStatus{models.ChoreStatusPending}})
			if err != nil {
				t.Fatal(err)
			}
			if len(tasks) != 1 || tasks[0].AssignedToUserID == nil || *tasks[0].AssignedToUserID == first {
				t.Fatalf("next occurrence did not rotate: %+v", tasks)
			}
			if err := service.SkipChore(ctx, tasks[0].ID, users[0].ID); err != nil {
				t.Fatal(err)
			}
			skipped, err := store.FindByID(ctx, tasks[0].ID)
			if err != nil || skipped.Status != models.ChoreStatusSkipped {
				t.Fatalf("skip: %+v %v", skipped, err)
			}

			if _, err := service.UpdateExistingChore(ctx, skipped, nil); !errors.Is(err, choreapp.ErrChoreAlreadyComplete) {
				t.Fatalf("updated a skipped chore: %v", err)
			}

			for _, invalidLimit := range []int{-1, 0, 101} {
				if _, err := service.History(ctx, invalidLimit, "", ""); err == nil {
					t.Fatalf("accepted invalid history limit %d", invalidLimit)
				}
			}
			history, err := service.History(ctx, 1, "", "")
			if err != nil || len(history.Items) != 1 || history.NextCursor == "" {
				t.Fatalf("history pagination: %+v %v", history, err)
			}

			nextPage, err := service.History(ctx, 1, history.NextCursor, "")
			if err != nil || len(nextPage.Items) != 1 || nextPage.Items[0].ID == history.Items[0].ID {
				t.Fatalf("history second page: %+v %v", nextPage, err)
			}

			broken, err := service.CreateChore(ctx, models.Chore{Name: "Invalid rule", CreatedByUserID: users[0].ID, DueDate: &due, RecurrenceType: models.RecurrenceDaily, RecurrenceValue: "{", RecurOnComplete: true}, nil)
			if err != nil {
				t.Fatal(err)
			}
			if err := service.CompleteChore(ctx, broken.ID, users[0].ID); err == nil {
				t.Fatal("invalid recurrence was accepted")
			}
			afterFailure, err := store.FindByID(ctx, broken.ID)
			if err != nil || afterFailure.Status != models.ChoreStatusPending || afterFailure.CompletedAt != nil {
				t.Fatalf("failed resolution was not rolled back: %+v %v", afterFailure, err)
			}
		})
	}
}

func fixedChoreTime() time.Time { return time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC) }

func TestChoreConcurrentCompletionContract(t *testing.T) {
	factories := map[string]func(*testing.T) (*choreapp.ChoreService, choreapp.ChoreRepository, models.User){
		"sqlite": func(t *testing.T) (*choreapp.ChoreService, choreapp.ChoreRepository, models.User) {
			database := testutil.NewTestDatabase(t)
			people := repository.NewUserRepository(database)
			user, err := people.Create(context.Background(), models.User{OIDCSubject: "concurrent", Email: "concurrent@example.com", Name: "Member", Role: models.RoleMember})
			if err != nil {
				t.Fatal(err)
			}
			store := repository.NewChoreRepository(database)
			return choreapp.NewChoreService(store, repository.NewChoreAssignmentRepository(database), people, repository.NewChoreSeriesRepository(database), choreapp.WithChoreClock(fixedChoreTime)), store, user
		},
		"fake": func(t *testing.T) (*choreapp.ChoreService, choreapp.ChoreRepository, models.User) {
			user := models.User{ID: "member", Name: "Member"}
			store := chorefakes.New([]models.User{user})
			return choreapp.NewChoreService(store, store.Assignments(), store.Users(), store.Series(), choreapp.WithChoreClock(fixedChoreTime)), store, user
		},
	}
	for name, factory := range factories {
		t.Run(name, func(t *testing.T) {
			service, store, user := factory(t)
			ctx := context.Background()
			task, err := service.CreateChore(ctx, models.Chore{Name: "Clean", CreatedByUserID: user.ID}, nil)
			if err != nil {
				t.Fatal(err)
			}
			results := make(chan error, 2)
			start := make(chan struct{})
			for range 2 {
				go func() { <-start; results <- service.CompleteChore(ctx, task.ID, user.ID) }()
			}
			close(start)
			success, alreadyResolved := 0, 0
			for range 2 {
				err := <-results
				switch {
				case err == nil:
					success++
				case errors.Is(err, choreapp.ErrChoreAlreadyComplete):
					alreadyResolved++
				default:
					t.Fatalf("completion error: %v", err)
				}
			}
			if success != 1 || alreadyResolved != 1 {
				t.Fatalf("success=%d already resolved=%d", success, alreadyResolved)
			}
			saved, err := store.FindByID(ctx, task.ID)
			if err != nil || saved.Status != models.ChoreStatusCompleted {
				t.Fatalf("saved chore: %+v %v", saved, err)
			}
		})
	}
}

func TestChoreFilteringContract(t *testing.T) {
	factories := map[string]func(*testing.T) choreFixture{
		"sqlite": func(t *testing.T) choreFixture {
			database := testutil.NewTestDatabase(t)
			user, err := repository.NewUserRepository(database).Create(context.Background(), models.User{OIDCSubject: "filters", Email: "filters@example.com", Name: "Member", Role: models.RoleMember})
			if err != nil {
				t.Fatal(err)
			}
			return choreFixture{store: repository.NewChoreRepository(database), user: user}
		},
		"fake": func(t *testing.T) choreFixture {
			user := models.User{ID: "member", Name: "Member"}
			return choreFixture{store: chorefakes.New([]models.User{user}), user: user}
		},
	}
	for name, factory := range factories {
		t.Run(name, func(t *testing.T) {
			fixture := factory(t)
			ctx := context.Background()
			seriesID := "series"
			firstDate := fixedChoreTime()
			secondDate := firstDate.AddDate(0, 0, 1)
			err := fixture.store.Transact(ctx, func(_ choreapp.ChoreRepository, _ choreapp.ChoreAssignmentRepository, _ choreapp.UserRepository, definitions choreapp.ChoreSeriesRepository) error {
				_, err := definitions.Create(ctx, models.ChoreSeries{ID: seriesID, Name: "Daily", CreatedByUserID: fixture.user.ID, RecurrenceType: models.RecurrenceDaily})
				return err
			})
			if err != nil {
				t.Fatal(err)
			}
			first, err := fixture.store.Create(ctx, models.Chore{Name: "First", CreatedByUserID: fixture.user.ID, AssignedToUserID: &fixture.user.ID, SeriesID: &seriesID, DueDate: &firstDate})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := fixture.store.Create(ctx, models.Chore{Name: "Duplicate pending", CreatedByUserID: fixture.user.ID, SeriesID: &seriesID}); err == nil {
				t.Fatal("accepted two outstanding chores in one series")
			}
			_, err = fixture.store.Create(ctx, models.Chore{Name: "Second", CreatedByUserID: fixture.user.ID, SeriesID: &seriesID, DueDate: &secondDate, Status: models.ChoreStatusCompleted})
			if err != nil {
				t.Fatal(err)
			}
			_, err = fixture.store.Create(ctx, models.Chore{Name: "Skipped", CreatedByUserID: fixture.user.ID, SeriesID: &seriesID, Status: models.ChoreStatusSkipped})
			if err != nil {
				t.Fatal(err)
			}
			tests := []struct {
				name   string
				filter choreapp.ChoreFilter
				want   int
			}{
				{"default excludes skipped", choreapp.ChoreFilter{}, 2},
				{"include history", choreapp.ChoreFilter{IncludeHistory: true}, 3},
				{"only history", choreapp.ChoreFilter{IncludeHistory: true, HistoryOnly: true}, 2},
				{"series", choreapp.ChoreFilter{SeriesID: &seriesID}, 2},
				{"assignee", choreapp.ChoreFilter{AssignedToUser: &fixture.user.ID}, 1},
				{"due by", choreapp.ChoreFilter{DueBefore: &firstDate}, 1},
				{"due from", choreapp.ChoreFilter{DueAfter: &secondDate}, 1},
				{"recurrence", choreapp.ChoreFilter{RecurrenceTypes: []models.RecurrenceType{models.RecurrenceDaily}}, 2},
				{"next in series", choreapp.ChoreFilter{OnlyNextPerSeries: true}, 1},
			}
			for _, test := range tests {
				t.Run(test.name, func(t *testing.T) {
					tasks, err := fixture.store.FindAll(ctx, test.filter)
					if err != nil || len(tasks) != test.want {
						t.Fatalf("got %d tasks, want %d: %v", len(tasks), test.want, err)
					}
					if test.name == "next in series" && tasks[0].ID != first.ID {
						t.Fatalf("got %s, want %s", tasks[0].ID, first.ID)
					}
				})
			}
			count, err := fixture.store.CountBySeries(ctx, seriesID)
			if err != nil || count != 3 {
				t.Fatalf("series count %d: %v", count, err)
			}
		})
	}
}
