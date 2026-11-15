package contracttests

import (
	"context"
	"errors"
	"testing"
	"time"

	app "github.com/bensuskins/family-hub/internal/application/displays"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
)

type displaySettingsFake struct{ item *app.Display }

func (f *displaySettingsFake) List(context.Context) ([]app.Display, error) {
	if f.item == nil {
		return []app.Display{}, nil
	}
	return []app.Display{*f.item}, nil
}
func (f *displaySettingsFake) FindByID(_ context.Context, id string) (app.Display, error) {
	if f.item == nil || f.item.ID != id {
		return app.Display{}, app.ErrDisplayNotFound
	}
	return *f.item, nil
}
func (f *displaySettingsFake) FindByTokenID(ctx context.Context, _ string) (app.Display, error) {
	return f.FindByID(ctx, "display")
}
func (f *displaySettingsFake) UpdateSections(_ context.Context, id string, sections []string) error {
	if f.item == nil || f.item.ID != id {
		return app.ErrDisplayNotFound
	}
	f.item.EnabledSections = sections
	return nil
}
func (f *displaySettingsFake) UpdateDetails(_ context.Context, id, name string) error {
	if f.item == nil || f.item.ID != id {
		return app.ErrDisplayNotFound
	}
	f.item.Name = name
	return nil
}
func (f *displaySettingsFake) UpdateCommutePreferences(_ context.Context, id string, p models.DisplayCommutePreferences) error {
	if f.item == nil || f.item.ID != id {
		return app.ErrDisplayNotFound
	}
	f.item.CommutePreferences = p
	return nil
}
func (f *displaySettingsFake) Revoke(_ context.Context, id string) error {
	if f.item == nil || f.item.ID != id {
		return app.ErrDisplayNotFound
	}
	f.item = nil
	return nil
}
func TestDisplaySettingsStoreContract(t *testing.T) {
	factories := map[string]func(*testing.T) app.SettingsStore{
		"fake": func(*testing.T) app.SettingsStore {
			return &displaySettingsFake{item: &app.Display{ID: "display", Name: "Kitchen"}}
		},
		"sqlite": func(t *testing.T) app.SettingsStore {
			db := testutil.NewTestDatabase(t)
			ctx := context.Background()
			user, err := repository.NewUserRepository(db).Create(ctx, models.User{OIDCSubject: "display-settings", Email: "display-settings@example.com", Name: "Admin", Role: models.RoleAdmin})
			if err != nil {
				t.Fatal(err)
			}
			store := repository.NewDisplayPairingRepository(db)
			if err := store.CreateChallenge(ctx, "display", "hash", time.Now().Add(5*time.Minute)); err != nil {
				t.Fatal(err)
			}
			if err := store.ApproveChallenge(ctx, "display", "hash", "Kitchen", user.ID); err != nil {
				t.Fatal(err)
			}
			if err := store.ClaimChallenge(ctx, "display", "hash", models.APIToken{TokenHash: "display-token"}); err != nil {
				t.Fatal(err)
			}
			return store
		},
	}
	for name, factory := range factories {
		t.Run(name, func(t *testing.T) {
			service := app.NewSettings(factory(t))
			ctx := context.Background()
			if err := service.UpdateSections(ctx, "display", []string{"calendar", "weather"}); err != nil {
				t.Fatal(err)
			}
			if err := service.UpdateDetails(ctx, "display", "Hall"); err != nil {
				t.Fatal(err)
			}
			p := models.DefaultDisplayCommutePreferences()
			p.HomeStationCode = "KEL"
			if err := service.UpdateCommutePreferences(ctx, "display", p); err != nil {
				t.Fatal(err)
			}
			item, err := service.FindByID(ctx, "display")
			if err != nil || item.Name != "Hall" || len(item.EnabledSections) != 2 || item.CommutePreferences.HomeStationCode != "KEL" {
				t.Fatalf("display=%+v err=%v", item, err)
			}
			if err := service.Revoke(ctx, "display"); err != nil {
				t.Fatal(err)
			}
			if _, err := service.FindByID(ctx, "display"); !errors.Is(err, app.ErrDisplayNotFound) {
				t.Fatalf("missing error=%v", err)
			}
		})
	}
}
