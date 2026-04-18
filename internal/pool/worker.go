package pool

import (
	"context"
	"time"
)

func (p *Pool) worker(ctx context.Context) {
	defer p.wg.Done()

	p.logger.Info("worker started")
	defer p.logger.Info("worker stopped")

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("worker canceled")
			return
		case job, ok := <-p.jobs:
			if !ok {
				p.logger.Info("job channel closed")
				return
			}

			started := time.Now()
			result := processJob(job, started)

			select {
			case <-ctx.Done():
				p.logger.Info("result dropped due to cancel")
				return
			case p.results <- result:
				p.logger.Info("job processed")
			}
		}
	}
}

func processJob(job URLJob, started time.Time) JobResult {
	duration := time.Since(started)
	if duration <= 0 {
		duration = time.Nanosecond
	}

	status := "processed"
	if job.URL == "" {
		status = "invalid"
	}

	return JobResult{
		JobID:    job.ID,
		Status:   status,
		Err:      nil,
		Duration: duration,
	}
}
