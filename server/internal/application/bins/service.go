package bins

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type Collection struct{ Date, Type, Name string }
type Schedule struct {
	Collections []Collection
	Enabled     bool
	UploadedAt  string
}
type Store interface {
	Load(context.Context) (Schedule, error)
	Save(context.Context, Schedule) error
}
type Parser interface {
	Parse(string, time.Time) ([]Collection, error)
}

var (
	ErrParse           = errors.New("invalid bin calendar")
	ErrSave            = errors.New("save bin schedule")
	ErrLoadSaved       = errors.New("load saved bin schedule")
	ErrLoad            = errors.New("load bin schedule")
	ErrUpdate          = errors.New("update bin reminders")
	ErrPushUnavailable = errors.New("push notifications unavailable")
	ErrNoCollections   = errors.New("no bin collections")
)

type Service struct {
	store         Store
	parser        Parser
	pushAvailable bool
	now           func() time.Time
}

func New(store Store, parser Parser, pushAvailable bool, now func() time.Time) *Service {
	return &Service{store: store, parser: parser, pushAvailable: pushAvailable, now: now}
}
func (service *Service) PushAvailable() bool { return service.pushAvailable }
func (service *Service) Get(ctx context.Context) (Schedule, error) {
	schedule, err := service.store.Load(ctx)
	if err != nil {
		return Schedule{}, fmt.Errorf("%w: %w", ErrLoad, err)
	}
	return copySchedule(schedule), nil
}
func (service *Service) Upload(ctx context.Context, data string) (Schedule, error) {
	collections, err := service.parser.Parse(data, service.now())
	if err != nil {
		return Schedule{}, fmt.Errorf("%w: %w", ErrParse, err)
	}
	if err := service.store.Save(ctx, Schedule{Collections: append([]Collection(nil), collections...), Enabled: service.pushAvailable, UploadedAt: service.now().UTC().Format(time.RFC3339)}); err != nil {
		return Schedule{}, fmt.Errorf("%w: %w", ErrSave, err)
	}
	schedule, err := service.store.Load(ctx)
	if err != nil {
		return Schedule{}, fmt.Errorf("%w: %w", ErrLoadSaved, err)
	}
	return copySchedule(schedule), nil
}
func (service *Service) SetEnabled(ctx context.Context, enabled bool) (Schedule, error) {
	if enabled && !service.pushAvailable {
		return Schedule{}, ErrPushUnavailable
	}
	if enabled {
		schedule, err := service.store.Load(ctx)
		if err != nil {
			return Schedule{}, fmt.Errorf("%w: %w", ErrLoad, err)
		}
		if len(schedule.Collections) == 0 {
			return Schedule{}, ErrNoCollections
		}
	}
	schedule, err := service.store.Load(ctx)
	if err != nil {
		return Schedule{}, fmt.Errorf("%w: %w", ErrUpdate, err)
	}
	schedule.Enabled = enabled
	if err := service.store.Save(ctx, schedule); err != nil {
		return Schedule{}, fmt.Errorf("%w: %w", ErrUpdate, err)
	}
	return copySchedule(schedule), nil
}
func copySchedule(schedule Schedule) Schedule {
	if schedule.Collections != nil {
		result := make([]Collection, len(schedule.Collections))
		copy(result, schedule.Collections)
		schedule.Collections = result
	}
	return schedule
}
