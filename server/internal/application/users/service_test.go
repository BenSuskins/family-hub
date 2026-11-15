package users

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeStore struct {
	user               User
	updateErr, findErr error
}

func (store *fakeStore) List(context.Context) ([]User, error)       { return []User{store.user}, nil }
func (store *fakeStore) Find(context.Context, string) (User, error) { return store.user, store.findErr }
func (store *fakeStore) SetRole(_ context.Context, _ string, role Role) error {
	store.user.Role = role
	return store.updateErr
}
func (store *fakeStore) SetAvatar(context.Context, string, string) error { return store.updateErr }
func (store *fakeStore) ClearAvatar(context.Context, string) error       { return store.updateErr }
func TestChangeRoleReportsReloadFailure(t *testing.T) {
	store := &fakeStore{findErr: errors.New("read failed")}
	_, err := New(store).ChangeRole(context.Background(), "user", RoleAdmin)
	if !errors.Is(err, ErrReload) {
		t.Fatalf("expected reload error, got %v", err)
	}
	if store.user.Role != RoleAdmin {
		t.Fatalf("role not written")
	}
}

func TestUserResultsDoNotExposeStoredOnboardingTime(t *testing.T) {
	original := time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC)
	storedTime := original
	store := &fakeStore{user: User{OnboardedAt: &storedTime}}
	service := New(store)
	user, err := service.Find(context.Background(), "one")
	if err != nil {
		t.Fatal(err)
	}
	*user.OnboardedAt = user.OnboardedAt.Add(time.Hour)
	if !store.user.OnboardedAt.Equal(original) {
		t.Fatal("find exposed stored time")
	}
	list, err := service.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	*list[0].OnboardedAt = list[0].OnboardedAt.Add(time.Hour)
	if !store.user.OnboardedAt.Equal(original) {
		t.Fatal("list exposed stored time")
	}
}
