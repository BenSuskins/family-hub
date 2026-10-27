package services

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
)

const binCalendarFixture = `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
UID:food-1
DTSTART;VALUE=DATE:20261028
SUMMARY:Food Waste
END:VEVENT
BEGIN:VEVENT
UID:garden-1
DTSTART;VALUE=DATE:20261028
SUMMARY:Garden Waste
END:VEVENT
BEGIN:VEVENT
UID:recycling-1
DTSTART;VALUE=DATE:20261104
SUMMARY:Mixed Recycling
END:VEVENT
BEGIN:VEVENT
UID:food-and-garden-1
DTSTART;VALUE=DATE:20261111
SUMMARY:Food Recycling and Garden Waste
END:VEVENT
BEGIN:VEVENT
UID:food-and-general-waste-1
DTSTART;VALUE=DATE:20261118
SUMMARY:Food Recycling and General Waste
END:VEVENT
BEGIN:VEVENT
UID:paper-and-card-waste-1
DTSTART;VALUE=DATE:20261125
SUMMARY:Paper and Card Waste
END:VEVENT
BEGIN:VEVENT
UID:unknown
DTSTART;VALUE=DATE:20261104
SUMMARY:Street sweeping
END:VEVENT
END:VCALENDAR`

func TestParseBinScheduleMapsUpcomingCollections(t *testing.T) {
	now := time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)
	data := strings.Replace(binCalendarFixture, "BEGIN:VEVENT\nUID:food-1", "BEGIN:VEVENT\nUID:old\nDTSTART;VALUE=DATE:20261020\nSUMMARY:General Waste\nEND:VEVENT\nBEGIN:VEVENT\nUID:food-1", 1)
	collections, err := ParseBinSchedule(data, now)
	if err != nil {
		t.Fatal(err)
	}
	want := []models.BinCollection{
		{Date: "2026-10-28", Type: "food_caddy", Name: "Food caddy"},
		{Date: "2026-10-28", Type: "garden", Name: "Garden bin"},
		{Date: "2026-11-04", Type: "mixed_recycling", Name: "Mixed recycling"},
		{Date: "2026-11-11", Type: "food_caddy", Name: "Food caddy"},
		{Date: "2026-11-11", Type: "garden", Name: "Garden bin"},
		{Date: "2026-11-18", Type: "food_caddy", Name: "Food caddy"},
		{Date: "2026-11-18", Type: "grey", Name: "Non-recyclable waste"},
		{Date: "2026-11-25", Type: "paper_card", Name: "Paper and card recycling"},
	}
	if len(collections) != len(want) {
		t.Fatalf("collections = %+v, want %+v", collections, want)
	}
	for index := range want {
		if collections[index] != want[index] {
			t.Errorf("collections[%d] = %+v, want %+v", index, collections[index], want[index])
		}
	}
}

func TestParseBinScheduleRejectsInvalidAndIrrelevantCalendar(t *testing.T) {
	for _, data := range []string{"not an ics file", "BEGIN:VCALENDAR\nEND:VCALENDAR", `BEGIN:VCALENDAR
BEGIN:VEVENT
DTSTART;VALUE=DATE:20990101
SUMMARY:Street sweeping
END:VEVENT
END:VCALENDAR`} {
		if _, err := ParseBinSchedule(data, time.Now()); err == nil {
			t.Fatalf("ParseBinSchedule(%q) succeeded", data)
		}
	}
}

func TestClassifyBinsMatchesWholeTermsAndExplicitWasteNames(t *testing.T) {
	tests := []struct {
		name    string
		summary string
		want    []string
	}{
		{name: "food waste", summary: "Food Waste", want: []string{"food_caddy"}},
		{name: "combined food and garden", summary: "Food Recycling and Garden Waste", want: []string{"food_caddy", "garden"}},
		{name: "explicit general waste", summary: "Food Recycling and General Waste", want: []string{"food_caddy", "grey"}},
		{name: "non-recyclable waste with paper", summary: "Non-Recyclable Waste and Paper and Card", want: []string{"paper_card", "grey"}},
		{name: "generic waste", summary: "Waste Collection", want: []string{"grey"}},
		{name: "partial food word", summary: "Seafood Festival", want: nil},
		{name: "partial card word", summary: "Cardigan Repair", want: nil},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bins := classifyBins(test.summary)
			got := make([]string, 0, len(bins))
			for _, bin := range bins {
				got = append(got, bin.Type)
			}
			if len(got) != len(test.want) {
				t.Fatalf("classifyBins(%q) types = %v, want %v", test.summary, got, test.want)
			}
			for index := range test.want {
				if got[index] != test.want[index] {
					t.Errorf("classifyBins(%q) types = %v, want %v", test.summary, got, test.want)
					break
				}
			}
		})
	}
}

func TestBinReminderMessageGroupsCollectionNames(t *testing.T) {
	collections := []models.BinCollection{
		{Name: "Food caddy"},
		{Name: "Garden bin"},
	}
	date := time.Date(2026, 10, 28, 0, 0, 0, 0, time.UTC)
	message := BinReminderMessage(collections, date)
	if message.Title != "Bins out tonight" || message.Body != "Food caddy and garden bin are due tomorrow, Wed 28 Oct." {
		t.Fatalf("unexpected reminder: %+v", message)
	}
	if strings.Contains(message.Body, "7am") || strings.Contains(message.Body, "7.00AM") {
		t.Fatalf("reminder includes a collection-time instruction: %q", message.Body)
	}
}

type fakeBinMessageSender struct {
	messages []PushMessage
}

func (sender *fakeBinMessageSender) SendMessage(_ context.Context, _ models.NotificationDevice, message PushMessage, _ string, _ time.Time) error {
	sender.messages = append(sender.messages, message)
	return nil
}

func TestBinReminderSendsOnceAtSevenToRegisteredDevices(t *testing.T) {
	ctx := context.Background()
	database := testutil.NewTestDatabase(t)
	users := repository.NewUserRepository(database)
	user, err := users.Create(ctx, models.User{OIDCSubject: "bin-reminders", Email: "bins@example.com", Name: "Family", Role: models.RoleAdmin})
	if err != nil {
		t.Fatal(err)
	}
	settings := repository.NewSettingsRepository(database)
	if err := SaveBinSchedule(ctx, settings, []models.BinCollection{
		{Date: "2026-10-28", Type: "food_caddy", Name: "Food caddy"},
		{Date: "2026-10-28", Type: "garden", Name: "Garden bin"},
	}, true, time.Now()); err != nil {
		t.Fatal(err)
	}
	devices := repository.NewNotificationRepository(database)
	if err := devices.Register(ctx, models.NotificationDevice{ID: "phone", UserID: user.ID, Token: "device-token", Environment: "sandbox"}); err != nil {
		t.Fatal(err)
	}
	sender := &fakeBinMessageSender{}
	service := NewBinReminderService(settings, devices, sender)
	location, err := time.LoadLocation("Europe/London")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 27, 19, 0, 0, 0, location)
	for range 2 {
		if err := service.Check(ctx, now); err != nil {
			t.Fatal(err)
		}
	}
	if len(sender.messages) != 1 {
		t.Fatalf("sent %d messages, want one", len(sender.messages))
	}
	if sender.messages[0].Body != "Food caddy and garden bin are due tomorrow, Wed 28 Oct." {
		t.Fatalf("unexpected reminder body: %q", sender.messages[0].Body)
	}
}
