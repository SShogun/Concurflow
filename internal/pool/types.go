package pool

import "time"

type URLJob struct {
	ID        string
	URL       string
	CreatedAt time.Time
}

type JobResult struct {
	JobID    string
	Status   string
	Err      error
	Duration time.Duration
}
