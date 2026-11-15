package identity_test

import (
	"context"
	"errors"
	"testing"
	"time"

	app "github.com/bensuskins/family-hub/internal/application/identity"
)

type administrationStoreFake struct {
	records     []app.TokenRecord
	createError error
}

func (store *administrationStoreFake) CreateManagedToken(_ context.Context, record app.TokenRecord) (app.TokenRecord, error) {
	if store.createError != nil {
		return app.TokenRecord{}, store.createError
	}
	record.ID = "token-id"
	store.records = append(store.records, record)
	return record, nil
}
func (store *administrationStoreFake) ListManagedTokens(context.Context) ([]app.TokenRecord, error) {
	return store.records, nil
}
func (store *administrationStoreFake) DeleteManagedToken(_ context.Context, id string) error {
	for index, record := range store.records {
		if record.ID == id {
			store.records = append(store.records[:index], store.records[index+1:]...)
			break
		}
	}
	return nil
}
func TestTokenAdministrationCreatesAndRevokesCredential(t *testing.T) {
	store := &administrationStoreFake{}
	service := app.NewTokenAdministration(store, func() (string, error) { return "raw-secret", nil }, func(secret string) string { return "hash:" + secret })
	created, err := service.Create(context.Background(), "Tablet", "user-id")
	if err != nil || created.ID != "token-id" || created.Name != "Tablet" || created.Token != "raw-secret" {
		t.Fatalf("created=%+v err=%v", created, err)
	}
	if len(store.records) != 1 || store.records[0].TokenHash != "hash:raw-secret" || store.records[0].Scope != "api" || store.records[0].CreatedByUserID != "user-id" {
		t.Fatalf("records=%+v", store.records)
	}
	records, err := service.List(context.Background())
	if err != nil || len(records) != 1 {
		t.Fatalf("records=%+v err=%v", records, err)
	}
	if err := service.Delete(context.Background(), created.ID); err != nil || len(store.records) != 0 {
		t.Fatalf("records=%+v err=%v", store.records, err)
	}
}
func TestTokenAdministrationDoesNotReturnSecretOnFailure(t *testing.T) {
	store := &administrationStoreFake{}
	service := app.NewTokenAdministration(store, func() (string, error) { return "", errors.New("random failed") }, func(string) string { return "hash" })
	result, err := service.Create(context.Background(), "name", "user")
	if !errors.Is(err, app.ErrGenerateToken) || result.Token != "" || len(store.records) != 0 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	store.createError = errors.New("storage failed")
	service = app.NewTokenAdministration(store, func() (string, error) { return "secret", nil }, func(string) string { return "hash" })
	result, err = service.Create(context.Background(), "name", "user")
	if !errors.Is(err, app.ErrCreateToken) || result.Token != "" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestTokenAdministrationCopiesExpirationOnList(t *testing.T) {
	expires := time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)
	original := expires
	store := &administrationStoreFake{records: []app.TokenRecord{{ID: "token", ExpiresAt: &expires}}}
	service := app.NewTokenAdministration(store, nil, nil)
	records, err := service.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	*records[0].ExpiresAt = expires.Add(time.Hour)
	if !store.records[0].ExpiresAt.Equal(original) {
		t.Fatalf("stored expiration changed: %v", store.records[0].ExpiresAt)
	}
}
