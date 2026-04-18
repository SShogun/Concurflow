package config

import "time"

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
