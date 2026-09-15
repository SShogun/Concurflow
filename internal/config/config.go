package config

import (
	"fmt"
	"time"
)

type Config struct {
	WorkerCount            int
	QueueDepth             int
	PipelineBufferSize     int
	MaxConcurrentDownloads int
	PerDownloadTimeout     time.Duration
	RunTimeout             time.Duration
}

func Default() Config {
	return Config{
		WorkerCount:            5,
		QueueDepth:             100,
		PipelineBufferSize:     10,
		MaxConcurrentDownloads: 3,
		PerDownloadTimeout:     10 * time.Second,
		RunTimeout:             1 * time.Minute,
	}
}

func (c Config) Validate() error {
	switch {
	case c.WorkerCount <= 0:
		return fmt.Errorf("worker count must be greater than zero")
	case c.QueueDepth < 0:
		return fmt.Errorf("queue depth cannot be negative")
	case c.PipelineBufferSize < 0:
		return fmt.Errorf("pipeline buffer size cannot be negative")
	case c.MaxConcurrentDownloads <= 0:
		return fmt.Errorf("max concurrent downloads must be greater than zero")
	case c.PerDownloadTimeout <= 0:
		return fmt.Errorf("per-download timeout must be greater than zero")
	case c.RunTimeout <= 0:
		return fmt.Errorf("run timeout must be greater than zero")
	default:
		return nil
	}
}
