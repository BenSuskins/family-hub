package contracttests

import (
	"context"
	app "github.com/bensuskins/family-hub/internal/application/chores"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
	"github.com/bensuskins/family-hub/internal/testutil/chorefakes"
	"testing"
	"time"
)

func TestChoreOverdueClockContract(t *testing.T) {
	factories := map[string]func(*testing.T) (app.ChoreRepository, string){
		"sqlite": func(t *testing.T) (app.ChoreRepository, string) {
			db := testutil.NewTestDatabase(t)
			user, err := repository.NewUserRepository(db).Create(context.Background(), models.User{OIDCSubject: "clock", Email: "clock@example.com", Name: "Clock", Role: models.RoleMember})
			if err != nil {
				t.Fatal(err)
			}
			return repository.NewChoreRepository(db), user.ID
		},
		"fake": func(t *testing.T) (app.ChoreRepository, string) {
			user := models.User{ID: "clock", Name: "Clock"}
			return chorefakes.New([]models.User{user}), user.ID
		},
	}
	for name, factory := range factories {
		t.Run(name, func(t *testing.T) {
			store, userID := factory(t)
			ctx := context.Background()
			now := time.Date(2001, time.June, 15, 12, 0, 0, 0, time.UTC)
			past := now.AddDate(0, 0, -1)
			future := now.AddDate(0, 0, 1)
			for _, entry := range []struct {
				name string
				due  *time.Time
			}{{"past", &past}, {"future", &future}} {
				if _, err := store.Create(ctx, models.Chore{Name: entry.name, CreatedByUserID: userID, DueDate: entry.due, Status: models.ChoreStatusPending}); err != nil {
					t.Fatal(err)
				}
			}
			items, err := store.FindOverdueChores(ctx, now)
			if err != nil {
				t.Fatal(err)
			}
			if len(items) != 1 || items[0].Name != "past" {
				t.Fatalf("overdue = %+v", items)
			}
		})
	}
}
