package commute

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/bensuskins/family-hub/internal/models"
)

const MaxCommuteTrains = 10

var (
	ErrCommuteUnavailable    = errors.New("train provider is not configured")
	ErrCommuteProvider       = errors.New("train provider is unavailable")
	ErrCommuteTimeout        = errors.New("train provider timed out")
	ErrInvalidCommuteRequest = errors.New("invalid commute route or count")
)

type Provider interface {
	FetchTrains(context.Context, string, string, int) (models.CommuteBoard, error)
}

func NormalizeStationCode(value string) (string, bool) {
	value = strings.ToUpper(strings.TrimSpace(value))
	if len(value) != 3 {
		return "", false
	}
	for _, character := range value {
		if character < 'A' || character > 'Z' {
			return "", false
		}
	}
	return value, true
}

type commuteCacheKey struct {
	origin      string
	destination string
	count       int
}

type commuteCacheEntry struct {
	board     models.CommuteBoard
	expiresAt time.Time
}

type CommuteService struct {
	client Provider
	now    func() time.Time
	mutex  sync.Mutex
	cache  map[commuteCacheKey]commuteCacheEntry
}

func New(client Provider, now func() time.Time) *CommuteService {
	if now == nil {
		now = time.Now
	}
	return &CommuteService{client: client, now: now, cache: make(map[commuteCacheKey]commuteCacheEntry)}
}

func (service *CommuteService) FetchTrains(ctx context.Context, origin, destination string, count int) (models.CommuteBoard, error) {
	origin, originValid := NormalizeStationCode(origin)
	destination, destinationValid := NormalizeStationCode(destination)
	if !originValid || !destinationValid || count < 1 || count > MaxCommuteTrains {
		return models.CommuteBoard{}, ErrInvalidCommuteRequest
	}
	key := commuteCacheKey{origin: origin, destination: destination, count: count}
	service.mutex.Lock()
	entry, exists := service.cache[key]
	service.mutex.Unlock()
	if exists && service.now().Before(entry.expiresAt) {
		return copyBoard(entry.board), nil
	}
	board, err := service.client.FetchTrains(ctx, origin, destination, count)
	if err != nil {
		return models.CommuteBoard{}, err
	}
	board.FetchedAt = service.now().UTC()
	service.mutex.Lock()
	service.cache[key] = commuteCacheEntry{board: copyBoard(board), expiresAt: board.FetchedAt.Add(30 * time.Second)}
	service.mutex.Unlock()
	return copyBoard(board), nil
}

func copyBoard(board models.CommuteBoard) models.CommuteBoard {
	if board.Trains == nil {
		return board
	}
	board.Trains = append([]models.CommuteTrain{}, board.Trains...)
	copyString := func(value *string) *string {
		if value == nil {
			return nil
		}
		result := *value
		return &result
	}
	for index := range board.Trains {
		train := &board.Trains[index]
		train.Platform = copyString(train.Platform)
		train.ScheduledDeparture = copyString(train.ScheduledDeparture)
		train.ExpectedDeparture = copyString(train.ExpectedDeparture)
		train.ScheduledArrival = copyString(train.ScheduledArrival)
		train.ExpectedArrival = copyString(train.ExpectedArrival)
		train.CancellationReason = copyString(train.CancellationReason)
		train.DelayReason = copyString(train.DelayReason)
	}
	return board
}
