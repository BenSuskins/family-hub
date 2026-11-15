package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

type lifecycleServer interface {
	Start() error
	Shutdown(context.Context) error
	Close() error
}
type worker struct {
	name     string
	interval time.Duration
	timeout  time.Duration
	run      func(context.Context) error
}

func serve(ctx context.Context, server lifecycleServer, workers []worker, shutdownTimeout time.Duration) error {
	workerContext, cancelWorkers := context.WithCancel(ctx)
	var group sync.WaitGroup
	for _, job := range workers {
		group.Add(1)
		go func() { defer group.Done(); runWorker(workerContext, job) }()
	}
	serverResult := make(chan error, 1)
	go func() { serverResult <- server.Start() }()
	var serveError error
	select {
	case <-ctx.Done():
	case err := <-serverResult:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveError = err
		}
	}
	cancelWorkers()
	shutdownContext, cancelShutdown := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancelShutdown()
	shutdownError := server.Shutdown(shutdownContext)
	if shutdownError != nil {
		_ = server.Close()
	}
	workersDone := make(chan struct{})
	go func() { group.Wait(); close(workersDone) }()
	select {
	case <-workersDone:
	case <-shutdownContext.Done():
		shutdownError = errors.Join(shutdownError, shutdownContext.Err())
	}
	return errors.Join(serveError, shutdownError)
}
func runWorker(ctx context.Context, job worker) {
	ticker := time.NewTicker(job.interval)
	defer ticker.Stop()
	for {
		if ctx.Err() != nil {
			return
		}
		operationContext := ctx
		cancel := func() {}
		if job.timeout > 0 {
			operationContext, cancel = context.WithTimeout(ctx, job.timeout)
		}
		err := job.run(operationContext)
		cancel()
		if err != nil && ctx.Err() == nil {
			slog.Error(job.name, "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
