package services

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/bensuskins/family-hub/internal/application/bins"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
)

func TestBinScheduleAdapterReadsLegacyValuesWithoutRewriting(t *testing.T) {
	settings := repository.NewSettingsRepository(testutil.NewTestDatabase(t))
	ctx := context.Background()
	legacy := `{"collections":null,"enabled":true,"uploaded_at":"legacy","extra":"keep"}`
	if err := settings.Set(ctx, binScheduleSetting, legacy); err != nil {
		t.Fatal(err)
	}
	schedule, err := NewBinScheduleAdapter(settings).Load(ctx)
	if err != nil || !schedule.Enabled || schedule.UploadedAt != "legacy" || schedule.Collections == nil || len(schedule.Collections) != 0 {
		t.Fatalf("load %+v, %v", schedule, err)
	}
	stored, err := settings.Get(ctx, binScheduleSetting)
	if err != nil || stored != legacy {
		t.Fatalf("legacy record rewritten: %q %v", stored, err)
	}
	if err := settings.Set(ctx, binScheduleSetting, "{"); err != nil {
		t.Fatal(err)
	}
	if _, err := NewBinScheduleAdapter(settings).Load(ctx); err == nil {
		t.Fatal("invalid stored JSON accepted")
	}
}
func TestBinCalendarParserMapsLegacyCollectionTypes(t *testing.T) {
	data := "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nBEGIN:VEVENT\r\nUID:food\r\nDTSTART;VALUE=DATE:20990101\r\nSUMMARY:Food Waste\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n"
	got, err := (BinCalendarParser{}).Parse(data, time.Date(2098, 12, 31, 0, 0, 0, 0, time.UTC))
	want := []bins.Collection{{Date: "2099-01-01", Type: "food_caddy", Name: "Food caddy"}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("parse %+v, %v", got, err)
	}
}
