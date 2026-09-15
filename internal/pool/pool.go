package pool

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/SShogun/Concurflow/internal/config"
)

var ErrClosed = errors.New("worker pool is shut down")

type Pool struct {
	cfg     config.Config
	logger  *slog.Logger
	jobs    chan URLJob
	results chan JobResult
	done    chan struct{}
	wg      sync.WaitGroup

	mu           sync.RWMutex
	startOnce    sync.Once
	shutdownOnce sync.Once
}

func New(cfg config.Config, logger *slog.Logger) (*Pool, error) {
	if cfg.WorkerCount <= 0 {
		return nil, fmt.Errorf("worker count must be greater than zero")
	}
	if cfg.QueueDepth < 0 {
		return nil, fmt.Errorf("queue depth cannot be negative")
	}
	if logger == nil {
		logger = slog.Default()
	}

	return &Pool{
		cfg:     cfg,
		logger:  logger,
		jobs:    make(chan URLJob, cfg.QueueDepth),
		results: make(chan JobResult, cfg.WorkerCount),
		done:    make(chan struct{}),
	}, nil
}

func (p *Pool) Start(ctx context.Context) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	select {
	case <-p.done:
		return
	default:
	}

	p.startOnce.Do(func() {
		for i := 0; i < p.cfg.WorkerCount; i++ {
			p.wg.Add(1)
			go p.worker(ctx)
		}
	})
}

func (p *Pool) Submit(ctx context.Context, job URLJob) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Check shutdown while jobs is guaranteed not to be closed under the read lock.
	select {
	case <-p.done:
		return ErrClosed
	default:
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-p.done:
		return ErrClosed
	case p.jobs <- job:
		return nil
	}
}

func (p *Pool) Results() <-chan JobResult {
	return p.results
}

func (p *Pool) Shutdown() {
	p.shutdownOnce.Do(func() {
		close(p.done)

		p.mu.Lock()
		close(p.jobs)
		p.mu.Unlock()

		p.wg.Wait()
		close(p.results)
	})
}
