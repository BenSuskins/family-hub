package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	ical "github.com/arran4/golang-ical"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
)

const binScheduleSetting = "bin_schedule"

func ParseBinSchedule(data string, now time.Time) ([]models.BinCollection, error) {
	calendar, err := ical.ParseCalendar(strings.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("parsing calendar: %w", err)
	}

	location, err := time.LoadLocation("Europe/London")
	if err != nil {
		return nil, err
	}
	today := now.In(location).Format("2006-01-02")
	collections := make([]models.BinCollection, 0, len(calendar.Events()))
	seen := map[string]bool{}
	for _, event := range calendar.Events() {
		summary := ""
		if property := event.GetProperty(ical.ComponentPropertySummary); property != nil {
			summary = strings.TrimSpace(property.Value)
		}
		bins := classifyBins(summary)
		if len(bins) == 0 {
			continue
		}
		start := event.GetProperty(ical.ComponentPropertyDtStart)
		if start == nil {
			continue
		}
		var date string
		if isAllDayProperty(start) {
			parsed, parseErr := event.GetAllDayStartAt()
			if parseErr != nil {
				continue
			}
			date = parsed.Format("2006-01-02")
		} else {
			parsed, parseErr := event.GetStartAt()
			if parseErr != nil {
				continue
			}
			date = parsed.In(location).Format("2006-01-02")
		}
		if date < today {
			continue
		}
		for _, bin := range bins {
			key := date + ":" + bin.Type
			if seen[key] {
				continue
			}
			seen[key] = true
			collections = append(collections, models.BinCollection{Date: date, Type: bin.Type, Name: bin.Name})
		}
	}
	if len(collections) == 0 {
		return nil, errors.New("calendar contains no upcoming bin collection dates")
	}
	sort.Slice(collections, func(first, second int) bool {
		if collections[first].Date != collections[second].Date {
			return collections[first].Date < collections[second].Date
		}
		return collections[first].Type < collections[second].Type
	})
	return collections, nil
}

func classifyBins(summary string) []models.BinCollection {
	value := strings.ToLower(summary)
	bins := make([]models.BinCollection, 0, 3)
	hasFood := strings.Contains(value, "food") || strings.Contains(value, "caddy")
	if hasFood {
		bins = append(bins, models.BinCollection{Type: "food_caddy", Name: "Food caddy"})
	}
	hasGarden := strings.Contains(value, "garden") || strings.Contains(value, "green waste")
	if hasGarden {
		bins = append(bins, models.BinCollection{Type: "garden", Name: "Garden bin"})
	}
	hasPaperOrCard := strings.Contains(value, "paper") || strings.Contains(value, "card")
	if hasPaperOrCard {
		bins = append(bins, models.BinCollection{Type: "paper_card", Name: "Paper and card recycling"})
	}
	hasMixedRecycling := strings.Contains(value, "mixed") || (strings.Contains(value, "recycl") && !hasPaperOrCard && !hasFood)
	if hasMixedRecycling {
		bins = append(bins, models.BinCollection{Type: "mixed_recycling", Name: "Mixed recycling"})
	}
	hasExplicitGeneralWaste := strings.Contains(value, "general") || strings.Contains(value, "refuse") || strings.Contains(value, "non-recyclable")
	hasRecognizedCollection := hasFood || hasGarden || hasPaperOrCard || hasMixedRecycling
	if hasExplicitGeneralWaste || (strings.Contains(value, "waste") && !hasRecognizedCollection) {
		bins = append(bins, models.BinCollection{Type: "grey", Name: "Non-recyclable waste"})
	}
	return bins
}

func LoadBinSchedule(ctx context.Context, settings repository.SettingsRepository) (models.BinSchedule, error) {
	schedule := models.BinSchedule{Collections: []models.BinCollection{}}
	value, err := settings.Get(ctx, binScheduleSetting)
	if errors.Is(err, sql.ErrNoRows) {
		return schedule, nil
	}
	if err != nil {
		return schedule, err
	}
	if err := json.Unmarshal([]byte(value), &schedule); err != nil {
		return schedule, err
	}
	if schedule.Collections == nil {
		schedule.Collections = []models.BinCollection{}
	}
	return schedule, nil
}

func SaveBinSchedule(ctx context.Context, settings repository.SettingsRepository, collections []models.BinCollection, enabled bool, now time.Time) error {
	value, err := json.Marshal(models.BinSchedule{Collections: collections, Enabled: enabled, UploadedAt: now.UTC().Format(time.RFC3339)})
	if err != nil {
		return err
	}
	return settings.Set(ctx, binScheduleSetting, string(value))
}

func UpdateBinScheduleEnabled(ctx context.Context, settings repository.SettingsRepository, enabled bool) (models.BinSchedule, error) {
	schedule, err := LoadBinSchedule(ctx, settings)
	if err != nil {
		return schedule, err
	}
	schedule.Enabled = enabled
	value, err := json.Marshal(schedule)
	if err != nil {
		return schedule, err
	}
	return schedule, settings.Set(ctx, binScheduleSetting, string(value))
}
