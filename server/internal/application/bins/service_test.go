package bins

import (
	"context"
	"errors"
	"testing"
	"time"
)

type storeFake struct {
	schedule                    Schedule
	loadErr, saveErr, updateErr error
	loads, saves, updates       int
	savedAt                     time.Time
	savedEnabled                bool
}

func (fake *storeFake) Load(context.Context) (Schedule, error) {
	fake.loads++
	return fake.schedule, fake.loadErr
}
func (fake *storeFake) Save(_ context.Context, schedule Schedule) error {
	fake.saves++
	fake.savedEnabled = schedule.Enabled
	fake.savedAt, _ = time.Parse(time.RFC3339, schedule.UploadedAt)
	if fake.saveErr != nil {
		return fake.saveErr
	}
	fake.schedule = schedule
	return nil
}

type parserFake struct {
	collections []Collection
	err         error
	parsedAt    time.Time
}

func (fake *parserFake) Parse(_ string, now time.Time) ([]Collection, error) {
	fake.parsedAt = now
	return fake.collections, fake.err
}
func TestUploadDoesNotSaveInvalidCalendar(t *testing.T) {
	store := &storeFake{}
	parser := &parserFake{err: errors.New("invalid")}
	service := New(store, parser, true, time.Now)
	_, err := service.Upload(context.Background(), "bad")
	if !errors.Is(err, ErrParse) || store.saves != 0 {
		t.Fatalf("upload: %v, saves %d", err, store.saves)
	}
}

func TestUploadSavesEnabledStateAndUsesTwoClockReadings(t *testing.T) {
	first := time.Date(2026, 1, 1, 23, 59, 59, 0, time.UTC)
	second := first.Add(time.Second)
	calls := 0
	clock := func() time.Time {
		calls++
		if calls == 1 {
			return first
		}
		return second
	}
	store := &storeFake{}
	parser := &parserFake{collections: []Collection{{Date: "2026-01-02", Type: "food", Name: "Food"}}}
	schedule, err := New(store, parser, true, clock).Upload(context.Background(), "calendar")
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 || parser.parsedAt != first || store.savedAt != second || !store.savedEnabled || store.loads != 1 {
		t.Fatalf("clock/operation order: calls=%d parsed=%v store=%+v", calls, parser.parsedAt, store)
	}
	schedule.Collections[0].Name = "changed"
	if store.schedule.Collections[0].Name != "Food" {
		t.Fatal("response aliases stored schedule")
	}
}
func TestSetEnabledPreconditionsAndWrites(t *testing.T) {
	for _, test := range []struct {
		name          string
		push, enabled bool
		collections   []Collection
		wantErr       error
		loads, saves  int
	}{
		{name: "unavailable", enabled: true, wantErr: ErrPushUnavailable},
		{name: "empty", push: true, enabled: true, wantErr: ErrNoCollections, loads: 1},
		{name: "enable", push: true, enabled: true, collections: []Collection{{Date: "2099-01-01"}}, loads: 2, saves: 1},
		{name: "disable", enabled: false, loads: 1, saves: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			store := &storeFake{schedule: Schedule{Collections: test.collections, UploadedAt: "original"}}
			schedule, err := New(store, &parserFake{}, test.push, time.Now).SetEnabled(context.Background(), test.enabled)
			if !errors.Is(err, test.wantErr) || store.loads != test.loads || store.saves != test.saves {
				t.Fatalf("error %v, loads %d, saves %d", err, store.loads, store.saves)
			}
			if err == nil && (schedule.Enabled != test.enabled || schedule.UploadedAt != "original") {
				t.Fatalf("schedule %+v", schedule)
			}
		})
	}
}
func TestUploadDistinguishesSaveAndReloadFailures(t *testing.T) {
	failure := errors.New("store failure")
	parser := &parserFake{collections: []Collection{{Name: "Food"}}}
	for _, test := range []struct {
		name  string
		store *storeFake
		want  error
	}{{"save", &storeFake{saveErr: failure}, ErrSave}, {"reload", &storeFake{loadErr: failure}, ErrLoadSaved}} {
		t.Run(test.name, func(t *testing.T) {
			_, err := New(test.store, parser, true, time.Now).Upload(context.Background(), "calendar")
			if !errors.Is(err, test.want) || !errors.Is(err, failure) {
				t.Fatalf("error %v", err)
			}
		})
	}
}

func TestSetEnabledSeparatesPreconditionAndUpdateFailures(t *testing.T) {
	failure := errors.New("storage failure")
	for _, test := range []struct {
		name     string
		enabled  bool
		store    *storeFake
		expected error
	}{
		{"precondition read", true, &storeFake{loadErr: failure}, ErrLoad},
		{"update read", false, &storeFake{loadErr: failure}, ErrUpdate},
		{"update write", true, &storeFake{schedule: Schedule{Collections: []Collection{{Name: "Food"}}}, saveErr: failure}, ErrUpdate},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := New(test.store, &parserFake{}, true, time.Now).SetEnabled(context.Background(), test.enabled)
			if !errors.Is(err, test.expected) || !errors.Is(err, failure) {
				t.Fatalf("error %v", err)
			}
		})
	}
}
