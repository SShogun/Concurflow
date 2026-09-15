package pool

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/SShogun/Concurflow/internal/config"
)

func TestShutdownReturnsWithoutResultConsumer(t *testing.T) {
	cfg := config.Default()
	cfg.WorkerCount = 1
	cfg.QueueDepth = 4
	p, err := New(cfg, nil)
	if err != nil {
		t.Fatalf("New returned an unexpected error: %v", err)
	}

	p.Start(context.Background())
	for i := 0; i < 4; i++ {
		if err := p.Submit(context.Background(), URLJob{ID: strconv.Itoa(i), URL: "https://example.com"}); err != nil {
			t.Fatalf("Submit returned an unexpected error: %v", err)
		}
	}

	done := make(chan struct{})
	go func() {
		p.Shutdown()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Shutdown blocked while results were not being consumed")
	}
}

func TestSubmitAfterShutdownReturnsErrClosed(t *testing.T) {
	cfg := config.Default()
	p, err := New(cfg, nil)
	if err != nil {
		t.Fatalf("New returned an unexpected error: %v", err)
	}
	p.Start(context.Background())
	p.Shutdown()

	err = p.Submit(context.Background(), URLJob{ID: "late", URL: "https://example.com"})
	if !errors.Is(err, ErrClosed) {
		t.Fatalf("expected ErrClosed, got %v", err)
	}
}

func TestConcurrentSubmitAndShutdown(t *testing.T) {
	cfg := config.Default()
	cfg.WorkerCount = 2
	cfg.QueueDepth = 2
	p, err := New(cfg, nil)
	if err != nil {
		t.Fatalf("New returned an unexpected error: %v", err)
	}
	p.Start(context.Background())

	const submitters = 32
	start := make(chan struct{})
	errs := make(chan error, submitters)
	var wg sync.WaitGroup
	wg.Add(submitters)
	for i := 0; i < submitters; i++ {
		go func(id int) {
			defer wg.Done()
			<-start
			errs <- p.Submit(context.Background(), URLJob{ID: strconv.Itoa(id), URL: "https://example.com"})
		}(i)
	}

	close(start)
	p.Shutdown()
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil && !errors.Is(err, ErrClosed) {
			t.Fatalf("unexpected submit error during shutdown: %v", err)
		}
	}
}

func TestNewRejectsInvalidWorkerCount(t *testing.T) {
	cfg := config.Default()
	cfg.WorkerCount = 0
	if _, err := New(cfg, nil); err == nil {
		t.Fatal("expected invalid worker count to fail")
	}
}
