package contracttests

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	app "github.com/bensuskins/family-hub/internal/application/scratchpad"
	domain "github.com/bensuskins/family-hub/internal/domain/scratchpad"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
)

type scratchpadContractFake struct {
	mutex  sync.Mutex
	value  domain.Snapshot
	userID string
}

func (fake *scratchpadContractFake) Get(ctx context.Context) (domain.Snapshot, error) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	if err := ctx.Err(); err != nil {
		return domain.Snapshot{}, err
	}
	result := fake.value
	result.Strokes = domain.CopyStrokes(result.Strokes)
	if result.Strokes == nil {
		result.Strokes = []domain.Stroke{}
	}
	return result, nil
}
func (fake *scratchpadContractFake) Save(ctx context.Context, strokes []domain.Stroke, userID string) (domain.Snapshot, error) {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()
	if err := ctx.Err(); err != nil {
		return domain.Snapshot{}, err
	}
	if userID != fake.userID {
		return domain.Snapshot{}, errors.New("foreign key")
	}
	fake.value = domain.Snapshot{Strokes: domain.CopyStrokes(strokes), UpdatedAt: time.Now().UTC()}
	result := fake.value
	result.Strokes = domain.CopyStrokes(result.Strokes)
	return result, nil
}
func TestScratchpadStoreContract(t *testing.T) {
	factories := map[string]func(*testing.T) (app.Store, string){
		"fake": func(t *testing.T) (app.Store, string) {
			return &scratchpadContractFake{value: domain.Snapshot{Strokes: []domain.Stroke{}}, userID: "member"}, "member"
		},
		"sqlite": func(t *testing.T) (app.Store, string) {
			db := testutil.NewTestDatabase(t)
			user, err := repository.NewUserRepository(db).Create(context.Background(), models.User{OIDCSubject: "scratch-contract", Email: "scratch@example.com", Name: "Member", Role: models.RoleMember})
			if err != nil {
				t.Fatal(err)
			}
			return repository.NewScratchpadRepository(db), user.ID
		},
	}
	for name, factory := range factories {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			store, user := factory(t)
			initial, err := store.Get(ctx)
			if err != nil || initial.Strokes == nil || len(initial.Strokes) != 0 {
				t.Fatalf("initial %+v %v", initial, err)
			}
			input := []domain.Stroke{{Color: "#3c654c", Points: []domain.Point{{X: .2, Y: .3}}}}
			saved, err := store.Save(ctx, input, user)
			if err != nil || saved.UpdatedAt.IsZero() {
				t.Fatalf("save %+v %v", saved, err)
			}
			input[0].Points[0].X = .9
			saved.Strokes[0].Points[0].Y = .9
			got, err := store.Get(ctx)
			if err != nil || got.Strokes[0].Points[0] != (domain.Point{X: .2, Y: .3}) {
				t.Fatalf("copy %+v %v", got, err)
			}
			got.Strokes[0].Points[0].X = .7
			again, err := store.Get(ctx)
			if err != nil || again.Strokes[0].Points[0].X != .2 {
				t.Fatalf("read copy %+v %v", again, err)
			}
			if _, err := store.Save(ctx, nil, "missing"); err == nil {
				t.Fatal("expected missing user error")
			}
			canceled, cancel := context.WithCancel(ctx)
			cancel()
			if _, err := store.Get(canceled); !errors.Is(err, context.Canceled) {
				t.Fatalf("canceled get %v", err)
			}
			if _, err := store.Save(canceled, nil, user); !errors.Is(err, context.Canceled) {
				t.Fatalf("canceled save %v", err)
			}
			if _, err := store.Save(ctx, nil, user); err != nil {
				t.Fatal(err)
			}
			empty, err := store.Get(ctx)
			if err != nil || empty.Strokes == nil || len(empty.Strokes) != 0 {
				t.Fatalf("null strokes %+v %v", empty, err)
			}
		})
	}
}
