package observability

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

var durationBuckets = []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}

type requestLabels struct {
	method string
	route  string
	status int
}

type requestSeries struct {
	count   uint64
	sum     float64
	buckets []uint64
}

type dependencyLabels struct {
	dependency string
	route      string
}

type dependencyContextKey struct{}

// RoutePattern returns the matched route template without request-specific path values.
func RoutePattern(request *http.Request) string {
	if routeContext := chi.RouteContext(request.Context()); routeContext != nil && routeContext.RoutePattern() != "" {
		return routeContext.RoutePattern()
	}
	return "unmatched"
}

// HTTPMetrics records bounded-cardinality request counts and latency histograms.
type HTTPMetrics struct {
	mutex              sync.RWMutex
	series             map[requestLabels]*requestSeries
	dependencyFailures map[dependencyLabels]uint64
}

func NewHTTPMetrics() *HTTPMetrics {
	return &HTTPMetrics{
		series:             make(map[requestLabels]*requestSeries),
		dependencyFailures: make(map[dependencyLabels]uint64),
	}
}

func (metrics *HTTPMetrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		started := time.Now()
		response := chimiddleware.NewWrapResponseWriter(writer, request.ProtoMajor)
		request = request.WithContext(context.WithValue(request.Context(), dependencyContextKey{}, metrics))
		next.ServeHTTP(response, request)

		route := RoutePattern(request)
		status := response.Status()
		if status == 0 {
			status = http.StatusOK
		}
		metrics.observe(requestLabels{
			method: metricMethod(request.Method),
			route:  route,
			status: status,
		}, time.Since(started).Seconds())
	})
}

// RecordDependencyFailure records a failure from an optional upstream service.
func RecordDependencyFailure(request *http.Request, dependency string) {
	metrics, ok := request.Context().Value(dependencyContextKey{}).(*HTTPMetrics)
	if !ok {
		return
	}
	route := RoutePattern(request)
	metrics.mutex.Lock()
	defer metrics.mutex.Unlock()
	metrics.dependencyFailures[dependencyLabels{dependency: dependency, route: route}]++
}

func (metrics *HTTPMetrics) ServeHTTP(writer http.ResponseWriter, _ *http.Request) {
	writer.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	writer.Header().Set("Cache-Control", "no-store")
	metrics.writePrometheus(writer)
}

func (metrics *HTTPMetrics) observe(labels requestLabels, duration float64) {
	metrics.mutex.Lock()
	defer metrics.mutex.Unlock()

	series := metrics.series[labels]
	if series == nil {
		series = &requestSeries{buckets: make([]uint64, len(durationBuckets))}
		metrics.series[labels] = series
	}
	series.count++
	series.sum += duration
	for index, bound := range durationBuckets {
		if duration <= bound {
			series.buckets[index]++
		}
	}
}

func (metrics *HTTPMetrics) writePrometheus(writer http.ResponseWriter) {
	metrics.mutex.RLock()
	labels := make([]requestLabels, 0, len(metrics.series))
	seriesByLabels := make(map[requestLabels]requestSeries, len(metrics.series))
	for key := range metrics.series {
		labels = append(labels, key)
		series := metrics.series[key]
		seriesByLabels[key] = requestSeries{count: series.count, sum: series.sum, buckets: append([]uint64(nil), series.buckets...)}
	}
	dependencyLabelsList := make([]dependencyLabels, 0, len(metrics.dependencyFailures))
	dependencyFailures := make(map[dependencyLabels]uint64, len(metrics.dependencyFailures))
	for key, count := range metrics.dependencyFailures {
		dependencyLabelsList = append(dependencyLabelsList, key)
		dependencyFailures[key] = count
	}
	metrics.mutex.RUnlock()

	sort.Slice(labels, func(left, right int) bool {
		if labels[left].route != labels[right].route {
			return labels[left].route < labels[right].route
		}
		if labels[left].method != labels[right].method {
			return labels[left].method < labels[right].method
		}
		return labels[left].status < labels[right].status
	})
	sort.Slice(dependencyLabelsList, func(left, right int) bool {
		if dependencyLabelsList[left].dependency != dependencyLabelsList[right].dependency {
			return dependencyLabelsList[left].dependency < dependencyLabelsList[right].dependency
		}
		return dependencyLabelsList[left].route < dependencyLabelsList[right].route
	})

	_, _ = fmt.Fprintln(writer, "# HELP familyhub_http_requests_total Total HTTP requests handled by the Family Hub API.")
	_, _ = fmt.Fprintln(writer, "# TYPE familyhub_http_requests_total counter")
	for _, key := range labels {
		series := seriesByLabels[key]
		_, _ = fmt.Fprintf(writer, "familyhub_http_requests_total{method=%q,route=%q,status=%q} %d\n", key.method, key.route, strconv.Itoa(key.status), series.count)
	}

	_, _ = fmt.Fprintln(writer, "# HELP familyhub_http_request_duration_seconds HTTP request duration in seconds.")
	_, _ = fmt.Fprintln(writer, "# TYPE familyhub_http_request_duration_seconds histogram")
	for _, key := range labels {
		series := seriesByLabels[key]
		for index, bound := range durationBuckets {
			_, _ = fmt.Fprintf(writer, "familyhub_http_request_duration_seconds_bucket{method=%q,route=%q,status=%q,le=%q} %d\n", key.method, key.route, strconv.Itoa(key.status), strconv.FormatFloat(bound, 'f', -1, 64), series.buckets[index])
		}
		_, _ = fmt.Fprintf(writer, "familyhub_http_request_duration_seconds_bucket{method=%q,route=%q,status=%q,le=\"+Inf\"} %d\n", key.method, key.route, strconv.Itoa(key.status), series.count)
		_, _ = fmt.Fprintf(writer, "familyhub_http_request_duration_seconds_sum{method=%q,route=%q,status=%q} %s\n", key.method, key.route, strconv.Itoa(key.status), strconv.FormatFloat(series.sum, 'f', -1, 64))
		_, _ = fmt.Fprintf(writer, "familyhub_http_request_duration_seconds_count{method=%q,route=%q,status=%q} %d\n", key.method, key.route, strconv.Itoa(key.status), series.count)
	}

	_, _ = fmt.Fprintln(writer, "# HELP familyhub_dependency_failures_total Total failed calls to optional upstream dependencies.")
	_, _ = fmt.Fprintln(writer, "# TYPE familyhub_dependency_failures_total counter")
	for _, key := range dependencyLabelsList {
		_, _ = fmt.Fprintf(writer, "familyhub_dependency_failures_total{dependency=%q,route=%q} %d\n", key.dependency, key.route, dependencyFailures[key])
	}
}

func metricMethod(method string) string {
	switch strings.ToUpper(method) {
	case http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions:
		return strings.ToUpper(method)
	default:
		return "OTHER"
	}
}
