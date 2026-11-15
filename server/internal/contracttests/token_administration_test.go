package contracttests

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	app "github.com/bensuskins/family-hub/internal/application/identity"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
)

type managedTokenFake struct {
	records []app.TokenRecord
	next    int
}

func (store *managedTokenFake) CreateManagedToken(_ context.Context, record app.TokenRecord) (app.TokenRecord, error) {
	for _, existing := range store.records {
		if existing.TokenHash == record.TokenHash || (record.ID != "" && existing.ID == record.ID) {
			return app.TokenRecord{}, errors.New("duplicate token")
		}
	}
	if record.ID == "" {
		store.next++
		record.ID = fmt.Sprint(store.next)
	}
	record.CreatedAt = time.Now()
	if record.ExpiresAt != nil {
		expiry := *record.ExpiresAt
		record.ExpiresAt = &expiry
	}
	store.records = append([]app.TokenRecord{record}, store.records...)
	return record, nil
}
func (store *managedTokenFake) ListManagedTokens(context.Context) ([]app.TokenRecord, error) {
	result := make([]app.TokenRecord, len(store.records))
	copy(result, store.records)
	for index, record := range result {
		if record.ExpiresAt != nil {
			expiry := *record.ExpiresAt
			result[index].ExpiresAt = &expiry
		}
	}
	return result, nil
}
func (store *managedTokenFake) DeleteManagedToken(_ context.Context, id string) error {
	for index, record := range store.records {
		if record.ID == id {
			store.records = append(store.records[:index], store.records[index+1:]...)
			break
		}
	}
	return nil
}
func TestTokenAdministrationStoreContract(t *testing.T) {
	factories := map[string]func(*testing.T) (app.TokenStore, string){
		"fake": func(*testing.T) (app.TokenStore, string) { return &managedTokenFake{}, "user" },
		"sqlite": func(t *testing.T) (app.TokenStore, string) {
			database := testutil.NewTestDatabase(t)
			user, err := repository.NewUserRepository(database).Create(context.Background(), models.User{OIDCSubject: "token-contract", Name: "Family", Role: models.RoleAdmin})
			if err != nil {
				t.Fatal(err)
			}
			return repository.NewAPITokenRepository(database), user.ID
		},
	}
	for name, factory := range factories {
		t.Run(name, func(t *testing.T) {
			store, user := factory(t)
			ctx := context.Background()
			expiration := time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)
			originalExpiration := expiration
			first, err := store.CreateManagedToken(ctx, app.TokenRecord{Name: "Tablet", TokenHash: "hash-1", Scope: "api", CreatedByUserID: user, ExpiresAt: &expiration})
			if err != nil {
				t.Fatal(err)
			}
			expiration = expiration.Add(time.Hour)
			if first.ID == "" || first.CreatedAt.IsZero() {
				t.Fatalf("created=%+v", first)
			}
			_, err = store.CreateManagedToken(ctx, app.TokenRecord{Name: "Duplicate", TokenHash: "hash-1", Scope: "api", CreatedByUserID: user})
			if err == nil {
				t.Fatal("duplicate hash accepted")
			}
			second, err := store.CreateManagedToken(ctx, app.TokenRecord{Name: "Phone", TokenHash: "hash-2", Scope: "display", CreatedByUserID: user})
			if err != nil {
				t.Fatal(err)
			}
			records, err := store.ListManagedTokens(ctx)
			if err != nil || len(records) != 2 {
				t.Fatalf("records=%+v err=%v", records, err)
			}
			found := map[string]app.TokenRecord{}
			for _, record := range records {
				found[record.ID] = record
			}
			if found[first.ID].ExpiresAt == nil || !found[first.ID].ExpiresAt.Equal(originalExpiration) {
				t.Fatalf("expiration=%v", found[first.ID].ExpiresAt)
			}
			*found[first.ID].ExpiresAt = expiration
			again, err := store.ListManagedTokens(ctx)
			if err != nil {
				t.Fatal(err)
			}
			for _, token := range again {
				if token.ID == first.ID && !token.ExpiresAt.Equal(originalExpiration) {
					t.Fatalf("stored expiration changed: %v", token.ExpiresAt)
				}
			}
			if found[first.ID].TokenHash != "hash-1" || found[first.ID].CreatedByUserID != user || found[second.ID].Scope != "display" {
				t.Fatalf("records=%+v", records)
			}
			if err = store.DeleteManagedToken(ctx, first.ID); err != nil {
				t.Fatal(err)
			}
			if err = store.DeleteManagedToken(ctx, "unknown"); err != nil {
				t.Fatal(err)
			}
			records, err = store.ListManagedTokens(ctx)
			if err != nil || len(records) != 1 || records[0].ID != second.ID {
				t.Fatalf("records=%+v err=%v", records, err)
			}
		})
	}
}
