package downloader

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/SShogun/Concurflow/internal/config"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestRunCancellationWhileWaitingForPermitReturns(t *testing.T) {
	started := make(chan struct{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case started <- struct{}{}:
		default:
		}
		<-r.Context().Done()
	}))
	defer server.Close()

	cfg := config.Default()
	cfg.MaxConcurrentDownloads = 1
	cfg.PerDownloadTimeout = 5 * time.Second
	dl := New(cfg, testLogger())

	ctx, cancel := context.WithCancel(context.Background())
	requests := make([]DownloadRequest, 8)
	for i := range requests {
		requests[i] = DownloadRequest{ID: i, URL: server.URL}
	}

	type runResult struct {
		results []DownloadResult
		err     error
	}
	done := make(chan runResult, 1)
	go func() {
		results, err := dl.Run(ctx, requests)
		done <- runResult{results: results, err: err}
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("first request never acquired the download permit")
	}

	cancel()

	select {
	case got := <-done:
		if !errors.Is(got.err, context.Canceled) {
			t.Fatalf("expected context cancellation, got %v", got.err)
		}
		if len(got.results) != len(requests) {
			t.Fatalf("expected %d results, got %d", len(requests), len(got.results))
		}
		for i, result := range got.results {
			if result.ID != i {
				t.Fatalf("result order changed: index %d has request id %d", i, result.ID)
			}
			if !errors.Is(result.Err, context.Canceled) {
				t.Fatalf("request %d: expected cancellation error, got %v", i, result.Err)
			}
		}
	case <-time.After(time.Second):
		t.Fatal("Run did not return after cancellation; possible semaphore or goroutine leak")
	}
}

func TestRunRespectsConcurrencyLimit(t *testing.T) {
	const limit = int32(2)
	const requestCount = 6

	var current atomic.Int32
	var maximum atomic.Int32
	started := make(chan struct{}, requestCount)
	release := make(chan struct{})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		now := current.Add(1)
		defer current.Add(-1)
		for {
			old := maximum.Load()
			if now <= old || maximum.CompareAndSwap(old, now) {
				break
			}
		}
		started <- struct{}{}
		<-release
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := config.Default()
	cfg.MaxConcurrentDownloads = int(limit)
	cfg.PerDownloadTimeout = 2 * time.Second
	dl := New(cfg, testLogger())

	requests := make([]DownloadRequest, requestCount)
	for i := range requests {
		requests[i] = DownloadRequest{ID: i, URL: server.URL}
	}

	done := make(chan error, 1)
	go func() {
		_, err := dl.Run(context.Background(), requests)
		done <- err
	}()

	for i := int32(0); i < limit; i++ {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("expected initial requests to start")
		}
	}

	select {
	case <-started:
		t.Fatal("more requests started than the configured concurrency limit")
	case <-time.After(50 * time.Millisecond):
	}

	close(release)
	if err := <-done; err != nil {
		t.Fatalf("Run returned an unexpected error: %v", err)
	}
	if got := maximum.Load(); got > limit {
		t.Fatalf("maximum concurrency was %d, want <= %d", got, limit)
	}
}

func TestRunPerDownloadTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()

	cfg := config.Default()
	cfg.PerDownloadTimeout = 25 * time.Millisecond
	dl := New(cfg, testLogger())

	results, err := dl.Run(context.Background(), []DownloadRequest{{ID: 1, URL: server.URL}})
	if err != nil {
		t.Fatalf("Run returned parent-context error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected one result, got %d", len(results))
	}
	if !errors.Is(results[0].Err, context.DeadlineExceeded) {
		t.Fatalf("expected per-download deadline error, got %v", results[0].Err)
	}
}

func TestRunRejectsInvalidConcurrency(t *testing.T) {
	cfg := config.Default()
	cfg.MaxConcurrentDownloads = 0
	dl := New(cfg, testLogger())

	if _, err := dl.Run(context.Background(), []DownloadRequest{{ID: 1, URL: "https://example.com"}}); err == nil {
		t.Fatal("expected invalid concurrency configuration to fail")
	}
}
