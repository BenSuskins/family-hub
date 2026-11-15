package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/bensuskins/family-hub/internal/config"
)

type lifecycleHTTPServer struct {
	server   *http.Server
	listener net.Listener
}

func (s *lifecycleHTTPServer) Start() error                       { return s.server.Serve(s.listener) }
func (s *lifecycleHTTPServer) Shutdown(ctx context.Context) error { return s.server.Shutdown(ctx) }
func (s *lifecycleHTTPServer) Close() error                       { return s.server.Close() }
func TestShutdownCancelsWorkersAndDrainsActiveRequests(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	release := make(chan struct{})
	workerStarted := make(chan struct{})
	workerStopped := make(chan struct{})
	server := &lifecycleHTTPServer{server: &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-release
		w.WriteHeader(http.StatusNoContent)
	})}, listener: listener}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- serve(ctx, server, []worker{{interval: time.Hour, run: func(ctx context.Context) error {
			close(workerStarted)
			<-ctx.Done()
			close(workerStopped)
			return ctx.Err()
		}}}, time.Second)
	}()
	<-workerStarted
	response := make(chan error, 1)
	go func() {
		result, err := http.Get("http://" + listener.Addr().String())
		if err == nil {
			result.Body.Close()
		}
		response <- err
	}()
	<-started
	cancel()
	select {
	case <-workerStopped:
	case <-time.After(time.Second):
		t.Fatal("worker was not cancelled")
	}
	select {
	case err := <-done:
		t.Fatalf("server did not drain active request: %v", err)
	default:
	}
	close(release)
	if err := <-response; err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
func TestShutdownDeadlineBoundsUncooperativeWorker(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := &lifecycleHTTPServer{server: &http.Server{}, listener: listener}
	started := make(chan struct{})
	release := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- serve(ctx, server, []worker{{interval: time.Hour, run: func(context.Context) error { close(started); <-release; return nil }}}, 50*time.Millisecond)
	}()
	<-started
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("shutdown error=%v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("shutdown exceeded deadline")
	}
	close(release)
}

type failingServer struct {
	started <-chan struct{}
	failure error
	closed  bool
}

func (s *failingServer) Start() error                   { <-s.started; return s.failure }
func (s *failingServer) Shutdown(context.Context) error { s.closed = true; return nil }
func (s *failingServer) Close() error                   { return nil }
func TestServerFailureCancelsWorkers(t *testing.T) {
	started := make(chan struct{})
	stopped := make(chan struct{})
	failure := errors.New("listener failed")
	server := &failingServer{started: started, failure: failure}
	err := serve(context.Background(), server, []worker{{interval: time.Hour, run: func(ctx context.Context) error { close(started); <-ctx.Done(); close(stopped); return ctx.Err() }}}, time.Second)
	if !errors.Is(err, failure) || !server.closed {
		t.Fatalf("error=%v shutdown=%v", err, server.closed)
	}
	select {
	case <-stopped:
	default:
		t.Fatal("worker did not stop")
	}
}

func TestCompositionRejectsMissingPushKey(t *testing.T) {
	configuration := config.Config{APNsKeyID: "key", APNsTeamID: "team", APNsTopic: "topic", APNsKeyPath: "/missing/family-hub-key"}
	_, _, err := compose(context.Background(), nil, configuration, time.Now)
	if err == nil {
		t.Fatal("expected missing push key to fail startup")
	}
}

func TestShutdownClosesRequestsAfterDeadline(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	finished := make(chan struct{})
	handler := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { close(started); <-r.Context().Done(); close(finished) })
	server := &lifecycleHTTPServer{server: &http.Server{Handler: handler}, listener: listener}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- serve(ctx, server, nil, 50*time.Millisecond) }()
	response := make(chan error, 1)
	go func() {
		result, err := http.Get("http://" + listener.Addr().String())
		if err == nil {
			result.Body.Close()
		}
		response <- err
	}()
	<-started
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("shutdown error=%v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("shutdown exceeded deadline")
	}
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("request was not closed")
	}
	<-response
}
func TestWorkerDoesNotRunAgainAfterCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	count := 0
	runWorker(ctx, worker{interval: time.Nanosecond, run: func(context.Context) error { count++; cancel(); return nil }})
	if count != 1 {
		t.Fatalf("worker runs=%d", count)
	}
}
