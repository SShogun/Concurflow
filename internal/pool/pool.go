package pool

import (
	"Concurflow/internal/config"
	"context"
	"log/slog"
	"sync"
)

type Pool struct {
	cfg     config.Config
	logger  *slog.Logger
	jobs    chan URLJob
	results chan JobResult
	wg      sync.WaitGroup
}

func New(cfg config.Config, logger *slog.Logger) *Pool {
	return &Pool{
		cfg:     cfg,
		logger:  logger,
		jobs:    make(chan URLJob, cfg.QueueDepth),
		results: make(chan JobResult, cfg.WorkerCount),
	}
}

func (p *Pool) Start(ctx context.Context) {
	for i := 0; i < p.cfg.WorkerCount; i++ {
		p.wg.Add(1)
		go p.worker(ctx)
	}
}

func (p *Pool) Submit(ctx context.Context, job URLJob) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case p.jobs <- job:
		return nil
	}
}

func (p *Pool) Results() <-chan JobResult {
	return p.results
}

func (p *Pool) Shutdown() {
	close(p.jobs)
	p.wg.Wait()
	close(p.results)
}
