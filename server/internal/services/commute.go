package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sort"
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

type HuxleyClient struct {
	baseURL string
	token   string
	client  *http.Client
}

func NewHuxleyClient(baseURL, token string, client *http.Client) *HuxleyClient {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	httpClient := *client
	httpClient.CheckRedirect = func(request *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }
	return &HuxleyClient{baseURL: strings.TrimRight(baseURL, "/"), token: strings.TrimSpace(token), client: &httpClient}
}

type huxleyBoard struct {
	GeneratedAt   time.Time     `json:"generatedAt"`
	TrainServices []huxleyTrain `json:"trainServices"`
}

type huxleyTrain struct {
	ServiceID               string                `json:"serviceID"`
	Operator                string                `json:"operator"`
	Platform                string                `json:"platform"`
	ScheduledDeparture      string                `json:"std"`
	ExpectedDeparture       string                `json:"etd"`
	IsCancelled             bool                  `json:"isCancelled"`
	DestinationCancelled    bool                  `json:"filterLocationCancelled"`
	CancellationReason      string                `json:"cancelReason"`
	DelayReason             string                `json:"delayReason"`
	SubsequentCallingPoints []huxleyCallingPoints `json:"subsequentCallingPoints"`
}

type huxleyCallingPoints struct {
	CallingPoint []huxleyCallingPoint `json:"callingPoint"`
}

type huxleyCallingPoint struct {
	Code        string `json:"crs"`
	Scheduled   string `json:"st"`
	Expected    string `json:"et"`
	IsCancelled bool   `json:"isCancelled"`
}

func (client *HuxleyClient) FetchTrains(ctx context.Context, originCode, destinationCode string, count int) (models.CommuteBoard, error) {
	originCode, originValid := normalizeCommuteStationCode(originCode)
	destinationCode, destinationValid := normalizeCommuteStationCode(destinationCode)
	if !originValid || !destinationValid || count < 1 || count > MaxCommuteTrains {
		return models.CommuteBoard{}, ErrInvalidCommuteRequest
	}
	if client.token == "" {
		return models.CommuteBoard{}, ErrCommuteUnavailable
	}
	origin := models.CommuteStation{Code: originCode}
	destination := models.CommuteStation{Code: destinationCode}
	endpoint, err := url.Parse(fmt.Sprintf("%s/departures/%s/to/%s/%d", client.baseURL, origin.Code, destination.Code, count))
	if err != nil || (endpoint.Scheme != "http" && endpoint.Scheme != "https") || endpoint.Host == "" || endpoint.User != nil || endpoint.RawQuery != "" || endpoint.Fragment != "" {
		return models.CommuteBoard{}, ErrCommuteProvider
	}
	query := endpoint.Query()
	query.Set("expand", "true")
	query.Set("timeWindow", "120")
	query.Set("accessToken", client.token)
	endpoint.RawQuery = query.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return models.CommuteBoard{}, ErrCommuteProvider
	}
	response, err := client.client.Do(request)
	if err != nil {
		return models.CommuteBoard{}, commuteRequestError(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return models.CommuteBoard{}, ErrCommuteProvider
	}
	const maximumBodyBytes = 2 * 1024 * 1024
	data, err := io.ReadAll(io.LimitReader(response.Body, maximumBodyBytes+1))
	if err != nil {
		return models.CommuteBoard{}, commuteRequestError(err)
	}
	var source huxleyBoard
	if len(data) > maximumBodyBytes || json.Unmarshal(data, &source) != nil || source.GeneratedAt.IsZero() {
		return models.CommuteBoard{}, ErrCommuteProvider
	}
	board := models.CommuteBoard{
		Origin: origin, Destination: destination,
		GeneratedAt: source.GeneratedAt, FetchedAt: time.Now().UTC(), Timezone: "Europe/London",
		Trains: make([]models.CommuteTrain, 0, len(source.TrainServices)),
	}
	for _, sourceTrain := range source.TrainServices {
		train := models.CommuteTrain{
			ServiceID: sourceTrain.ServiceID, Operator: sourceTrain.Operator,
			Platform:           optionalCommuteText(sourceTrain.Platform),
			ScheduledDeparture: commuteClock(sourceTrain.ScheduledDeparture),
			ExpectedDeparture:  commuteEstimate(sourceTrain.ExpectedDeparture, sourceTrain.ScheduledDeparture),
			CancellationReason: optionalCommuteText(sourceTrain.CancellationReason),
			DelayReason:        optionalCommuteText(sourceTrain.DelayReason),
		}
		cancelled := sourceTrain.IsCancelled || sourceTrain.DestinationCancelled
		arrivalEstimate := ""
		for _, group := range sourceTrain.SubsequentCallingPoints {
			for _, point := range group.CallingPoint {
				if point.Code == destination.Code {
					train.ScheduledArrival = commuteClock(point.Scheduled)
					train.ExpectedArrival = commuteEstimate(point.Expected, point.Scheduled)
					arrivalEstimate = point.Expected
					cancelled = cancelled || point.IsCancelled
				}
			}
		}
		switch {
		case cancelled || strings.EqualFold(sourceTrain.ExpectedDeparture, "Cancelled") || strings.EqualFold(arrivalEstimate, "Cancelled"):
			train.Status = "cancelled"
			train.ExpectedDeparture, train.ExpectedArrival = nil, nil
		case strings.EqualFold(sourceTrain.ExpectedDeparture, "Delayed") || strings.EqualFold(arrivalEstimate, "Delayed") ||
			commuteTimeIsLater(train.ExpectedDeparture, train.ScheduledDeparture) || commuteTimeIsLater(train.ExpectedArrival, train.ScheduledArrival):
			train.Status = "delayed"
		case train.ExpectedDeparture != nil:
			train.Status = "on_time"
		default:
			train.Status = "unknown"
		}
		board.Trains = append(board.Trains, train)
	}
	sort.SliceStable(board.Trains, func(first, second int) bool {
		return commuteDepartureOrder(board.Trains[first].ScheduledDeparture, board.GeneratedAt) <
			commuteDepartureOrder(board.Trains[second].ScheduledDeparture, board.GeneratedAt)
	})
	if len(board.Trains) > count {
		board.Trains = board.Trains[:count]
	}
	return board, nil
}

func normalizeCommuteStationCode(value string) (string, bool) {
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

func commuteRequestError(err error) error {
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	var networkError net.Error
	if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &networkError) && networkError.Timeout()) {
		return ErrCommuteTimeout
	}
	// HTTP errors can contain the request URL and its access token.
	return ErrCommuteProvider
}

func optionalCommuteText(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func commuteEstimate(estimate, scheduled string) *string {
	if strings.EqualFold(strings.TrimSpace(estimate), "On time") {
		return commuteClock(scheduled)
	}
	return commuteClock(estimate)
}

func commuteClock(value string) *string {
	value = strings.TrimSpace(value)
	if len(value) != 5 {
		return nil
	}
	if _, err := time.Parse("15:04", value); err != nil {
		return nil
	}
	return &value
}

func commuteMinutes(value string) int {
	return int(value[0]-'0')*600 + int(value[1]-'0')*60 + int(value[3]-'0')*10 + int(value[4]-'0')
}

func commuteTimeIsLater(expected, scheduled *string) bool {
	if expected == nil || scheduled == nil {
		return false
	}
	difference := (commuteMinutes(*expected) - commuteMinutes(*scheduled) + 1440) % 1440
	return difference > 0 && difference < 720
}

func commuteDepartureOrder(scheduled *string, generatedAt time.Time) int {
	if scheduled == nil {
		return 1440
	}
	// Boards span two hours. The nearest clock occurrence keeps delayed trains
	// before midnight ahead of departures after midnight.
	location, err := time.LoadLocation("Europe/London")
	if err == nil {
		generatedAt = generatedAt.In(location)
	}
	currentMinute := generatedAt.Hour()*60 + generatedAt.Minute()
	return (commuteMinutes(*scheduled)-currentMinute+2160)%1440 - 720
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
	client *HuxleyClient
	now    func() time.Time
	mutex  sync.Mutex
	cache  map[commuteCacheKey]commuteCacheEntry
}

func NewCommuteService(client *HuxleyClient, now func() time.Time) *CommuteService {
	if now == nil {
		now = time.Now
	}
	return &CommuteService{client: client, now: now, cache: make(map[commuteCacheKey]commuteCacheEntry)}
}

func (service *CommuteService) FetchTrains(ctx context.Context, origin, destination string, count int) (models.CommuteBoard, error) {
	origin, originValid := normalizeCommuteStationCode(origin)
	destination, destinationValid := normalizeCommuteStationCode(destination)
	if !originValid || !destinationValid || count < 1 || count > MaxCommuteTrains {
		return models.CommuteBoard{}, ErrInvalidCommuteRequest
	}
	key := commuteCacheKey{origin: origin, destination: destination, count: count}
	service.mutex.Lock()
	entry, exists := service.cache[key]
	service.mutex.Unlock()
	if exists && service.now().Before(entry.expiresAt) {
		return entry.board, nil
	}
	board, err := service.client.FetchTrains(ctx, origin, destination, count)
	if err != nil {
		return models.CommuteBoard{}, err
	}
	board.FetchedAt = service.now().UTC()
	service.mutex.Lock()
	service.cache[key] = commuteCacheEntry{board: board, expiresAt: board.FetchedAt.Add(30 * time.Second)}
	service.mutex.Unlock()
	return board, nil
}
